package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"tg/internal/tdlib"
)

// serveIPC opens the Unix socket that `tg send`/`tg ask` connect to so they can
// reuse the daemon's single Telegram session instead of opening their own.
func (d *daemon) serveIPC() error {
	path, err := SocketPath()
	if err != nil {
		return err
	}
	_ = os.Remove(path) // clear a stale socket from a previous run

	listener, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	// Restrict to the owner: a Unix socket needs write permission to connect, so
	// 0600 means only our user can route send/ask through the daemon.
	if err := os.Chmod(path, 0o600); err != nil {
		listener.Close()
		return err
	}
	fmt.Println("ipc socket:", path)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go d.handleIPC(conn)
		}
	}()
	return nil
}

func (d *daemon) handleIPC(conn net.Conn) {
	defer conn.Close()

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return
	}

	var req ipcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		writeIPC(conn, ipcResponse{Error: "bad request"})
		return
	}
	fmt.Printf("[ipc] %s -> %s\n", req.Op, req.To)

	switch req.Op {
	case "send":
		chatID, _, err := d.resolveChat(req.To)
		if err != nil {
			writeIPC(conn, ipcResponse{Error: err.Error()})
			return
		}
		msgID, err := tdlib.SendTextMessage(d.tdjson, d.clientID, chatID, req.Text)
		if err != nil {
			writeIPC(conn, ipcResponse{Error: err.Error()})
			return
		}
		writeIPC(conn, ipcResponse{OK: true, MessageID: msgID, ChatID: chatID})

	case "sendfile":
		chatID, _, err := d.resolveChat(req.To)
		if err != nil {
			writeIPC(conn, ipcResponse{Error: err.Error()})
			return
		}
		// Fire-and-forget: don't WaitForSendCompletion here. The completion
		// arrives as an unsolicited update that the daemon's main receive loop
		// would consume, so waiting on it would race/hang. The daemon stays
		// alive, so the upload finishes in the background regardless.
		_, label, err := tdlib.SendLocalFileMessage(d.tdjson, d.clientID, chatID, req.Path, req.Text)
		if err != nil {
			writeIPC(conn, ipcResponse{Error: err.Error()})
			return
		}
		writeIPC(conn, ipcResponse{OK: true, ChatID: chatID, Label: label})

	case "ask":
		chatID, userID, err := d.resolveChat(req.To)
		if err != nil {
			writeIPC(conn, ipcResponse{Error: err.Error()})
			return
		}
		// Empty text = `tg chat` "just listen" mode: don't send, only wait.
		if req.Text != "" {
			if _, err := tdlib.SendTextMessage(d.tdjson, d.clientID, chatID, req.Text); err != nil {
				writeIPC(conn, ipcResponse{Error: err.Error()})
				return
			}
		}

		replyCh := make(chan string, 1)
		d.mu.Lock()
		d.pendingAsk[userID] = append(d.pendingAsk[userID], replyCh)
		d.mu.Unlock()

		// Detect the client disconnecting so a dead `ask` doesn't later swallow
		// one of the user's bridge messages.
		clientGone := make(chan struct{})
		go func() {
			buf := make([]byte, 1)
			_, _ = conn.Read(buf)
			close(clientGone)
		}()

		select {
		case reply := <-replyCh:
			writeIPC(conn, ipcResponse{OK: true, Reply: reply})
		case <-clientGone:
			d.removePending(userID, replyCh)
		}

	default:
		writeIPC(conn, ipcResponse{Error: "unknown op: " + req.Op})
	}
}

func writeIPC(conn net.Conn, resp ipcResponse) {
	b, _ := json.Marshal(resp)
	_, _ = conn.Write(append(b, '\n'))
}

// resolveChat turns "@username" or a numeric id into a chat id (and user id).
func (d *daemon) resolveChat(to string) (chatID, userID int64, err error) {
	to = strings.TrimSpace(to)
	if id, e := strconv.ParseInt(to, 10, 64); e == nil {
		return id, id, nil // private chat id == user id in TDLib
	}
	uid, e := tdlib.ResolveUserIdentifierByUsername(d.tdjson, d.clientID, to)
	if e != nil {
		return 0, 0, e
	}
	cid, e := tdlib.CreatePrivateChat(d.tdjson, d.clientID, uid)
	if e != nil {
		cid = uid
	}
	return cid, uid, nil
}

// resolvePendingAsk delivers text to the oldest outstanding `tg ask` for userID,
// returning true if one was waiting.
func (d *daemon) resolvePendingAsk(userID int64, text string) bool {
	d.mu.Lock()
	queue := d.pendingAsk[userID]
	if len(queue) == 0 {
		d.mu.Unlock()
		return false
	}
	ch := queue[0]
	d.pendingAsk[userID] = queue[1:]
	d.mu.Unlock()

	ch <- text
	return true
}

func (d *daemon) removePending(userID int64, ch chan string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	queue := d.pendingAsk[userID]
	for i, c := range queue {
		if c == ch {
			d.pendingAsk[userID] = append(queue[:i], queue[i+1:]...)
			return
		}
	}
}

// replyText extracts a user's reply for `tg ask`, matching standalone behavior
// (text, else caption, else a [type] tag for media).
func replyText(c tdlib.Content) string {
	if c.Type == "messageText" {
		return c.Text.Text
	}
	if t := c.CaptionOrText(); t != "" {
		return t
	}
	return "[" + c.Type + "]"
}
