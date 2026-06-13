package agent

import (
	"fmt"
	"strings"
	"time"

	"tg/internal/tdlib"
)

// inboxMessage is one message from a non-allow-listed sender awaiting triage.
type inboxMessage struct {
	sender string
	text   string
}

// handleNonAllowlisted auto-replies (at most hourly per sender) and buffers a
// stranger's message for the next triage pass.
func (d *daemon) handleNonAllowlisted(msg tdlib.Message) {
	text := replyText(msg.Content)
	sender := d.senderName(msg.SenderID.UserID)

	d.mu.Lock()
	d.inbox = append(d.inbox, inboxMessage{sender: sender, text: text})
	last := d.lastAutoReply[msg.SenderID.UserID]
	shouldReply := d.settings.AutoReplyEnabled && d.settings.AutoReply != "" && time.Since(last) > time.Hour
	if shouldReply {
		d.lastAutoReply[msg.SenderID.UserID] = time.Now()
	}
	d.mu.Unlock()

	if shouldReply {
		d.send(msg.ChatID, d.settings.AutoReply)
	}
}

func (d *daemon) senderName(userID int64) string {
	d.mu.Lock()
	if cached, ok := d.nameCache[userID]; ok {
		d.mu.Unlock()
		return cached
	}
	d.mu.Unlock()

	name, err := tdlib.FetchUserDisplayName(d.tdjson, d.clientID, userID)
	if err != nil || name == "" {
		name = fmt.Sprintf("user:%d", userID)
	}
	d.mu.Lock()
	d.nameCache[userID] = name
	d.mu.Unlock()
	return name
}

// runTriageLoop periodically triages buffered stranger messages and DMs the main
// user a digest of the important ones (nothing if none).
func (d *daemon) runTriageLoop() {
	interval := time.Duration(d.settings.Triage.EveryMinutes) * time.Minute
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	fmt.Printf("[triage] enabled: every %s, agent=%s, dir=%s\n", interval, d.settings.Triage.Agent.normalize(), d.settings.Triage.Dir)

	for range ticker.C {
		d.runTriageOnce()
	}
}

func (d *daemon) runTriageOnce() {
	d.mu.Lock()
	msgs := d.inbox
	d.inbox = nil
	d.mu.Unlock()
	if len(msgs) == 0 {
		return
	}

	var b strings.Builder
	b.WriteString("You are triaging Telegram messages received for the owner while they were away.\n")
	b.WriteString("Decide which are IMPORTANT enough to notify them now (urgent, personal, time-sensitive, ")
	b.WriteString("or someone clearly needing a reply). Ignore spam, promotions, automated/bot noise, and trivial chatter.\n")
	b.WriteString("If NONE are important, reply with exactly: NONE\n")
	b.WriteString("Otherwise reply with a short bullet list, one per important message: '• <sender>: <one-line why it matters>'.\n")
	b.WriteString("Do not take any actions or run any commands.\n\nMessages:\n")
	for i, m := range msgs {
		fmt.Fprintf(&b, "%d. From %s: %s\n", i+1, m.sender, snippet(m.text, 300))
	}

	res, err := RunAgent(d.settings.Triage.Agent, d.settings.Triage.Dir, b.String(), "", RoleRead)
	if err != nil {
		fmt.Printf("[triage] error: %v\n", err)
		return
	}

	// Important items are bullet lines; ignore any preamble/NONE wording.
	var bullets []string
	for _, line := range strings.Split(res.Text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "•") || strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			bullets = append(bullets, t)
		}
	}
	if len(bullets) == 0 {
		fmt.Printf("[triage] %d message(s), none important\n", len(msgs))
		return
	}

	if d.mainChatID != 0 {
		d.send(d.mainChatID, "📨 Messages worth your attention:\n"+strings.Join(bullets, "\n"))
		fmt.Printf("[triage] %d message(s), digest sent to main user\n", len(msgs))
	} else {
		fmt.Printf("[triage] %d important but no main_user configured to notify\n", len(msgs))
	}
}
