package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tg/internal/tdlib"
)

// uploadsSubdir is where files sent through the bridge are saved, inside the
// active project so the (sandboxed) agent can read them.
const uploadsSubdir = "tg-uploads"

// handleIncomingFile downloads a file an allow-listed user sent into the active
// project and hands it to the agent, using the caption as the instruction.
func (d *daemon) handleIncomingFile(st *userState, msg tdlib.Message) {
	file, fileName, label, ok := msg.Content.MediaFile()
	if !ok {
		d.send(st.chatID, "I can only handle text and files here.")
		return
	}

	d.mu.Lock()
	loc, dir, busy, chatID := st.location, st.locationPath, st.busy, st.chatID
	if loc != "" && !busy {
		st.busy = true // hold through download + the agent run
	}
	d.mu.Unlock()

	if loc == "" {
		d.send(chatID, "Pick a location first (send 'locations'), then send the file — it needs a project to live in.")
		return
	}
	if busy {
		d.send(chatID, "⏳ Still working — one moment, then resend the file.")
		return
	}

	caption := msg.Content.CaptionOrText()
	if fileName == "" {
		fileName = label
	}
	d.send(chatID, "📎 Downloading "+fileName+"…")

	go func() {
		localPath, err := tdlib.DownloadFile(d.tdjson, d.clientID, file.ID, 10*time.Minute)
		if err != nil {
			d.releaseBusy(st)
			d.send(chatID, "⚠️ Couldn't download the file: "+err.Error())
			return
		}
		rel := filepath.Join(uploadsSubdir, sanitizeFilename(fileName))
		if err := copyInto(localPath, filepath.Join(dir, rel)); err != nil {
			d.releaseBusy(st)
			d.send(chatID, "⚠️ Couldn't save the file: "+err.Error())
			return
		}

		d.send(chatID, "📎 Saved to "+rel+" — passing it to the agent…")
		d.dispatchAgentTurn(st, fileTurnPrompt(caption, rel, label)) // runAgent clears busy
	}()
}

func (d *daemon) releaseBusy(st *userState) {
	d.mu.Lock()
	st.busy = false
	d.mu.Unlock()
}

// dispatchAgentTurn runs a message through the agent with confirm-aware routing
// (plan-first for the confirm role).
func (d *daemon) dispatchAgentTurn(st *userState, text string) {
	d.mu.Lock()
	role := st.effectiveRole
	d.mu.Unlock()
	if role == RoleConfirm {
		d.runAgent(st, text, RoleRead, true)
		return
	}
	d.runAgent(st, text, role, false)
}

func fileTurnPrompt(caption, relPath, label string) string {
	caption = strings.TrimSpace(caption)
	if caption != "" {
		return caption + "\n\n(I attached a " + label + " — it's saved at `" + relPath + "` in this project. Read it from there.)"
	}
	return "I've sent you a " + label + ", saved at `" + relPath + "` in this project. Take a look."
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	return strings.NewReplacer("/", "_", "\\", "_", "\n", " ", "\r", " ").Replace(name)
}

func copyInto(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
