package agent

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"time"
)

// The daemon owns the single Telegram session. When it's running, `tg send`/
// `tg ask` route their request through this Unix socket instead of opening their
// own client (which would deadlock on the session lock). When it's absent, the
// commands fall back to talking to Telegram directly.

const socketName = "daemon.sock"

func SocketPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, socketName), nil
}

type ipcRequest struct {
	Op   string `json:"op"`             // "send" | "ask" | "sendfile"
	To   string `json:"to"`             // @username or numeric chat id
	Text string `json:"text"`           // message body / question / caption
	Path string `json:"path,omitempty"` // local file path (sendfile)
}

type ipcResponse struct {
	OK        bool   `json:"ok"`
	MessageID int64  `json:"message_id,omitempty"`
	ChatID    int64  `json:"chat_id,omitempty"`
	Reply     string `json:"reply,omitempty"`
	Label     string `json:"label,omitempty"`
	Error     string `json:"error,omitempty"`
}

func dialDaemon() (net.Conn, bool) {
	path, err := SocketPath()
	if err != nil {
		return nil, false
	}
	conn, err := net.DialTimeout("unix", path, 2*time.Second)
	if err != nil {
		return nil, false
	}
	return conn, true
}

func roundTrip(req ipcRequest, readDeadline time.Time) (ipcResponse, bool, error) {
	conn, ok := dialDaemon()
	if !ok {
		return ipcResponse{}, false, nil // no daemon → caller falls back
	}
	defer conn.Close()

	line, err := json.Marshal(req)
	if err != nil {
		return ipcResponse{}, true, err
	}
	if _, err := conn.Write(append(line, '\n')); err != nil {
		return ipcResponse{}, true, err
	}

	if !readDeadline.IsZero() {
		_ = conn.SetReadDeadline(readDeadline)
	}
	respLine, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && len(respLine) == 0 {
		return ipcResponse{}, true, err
	}

	var resp ipcResponse
	if err := json.Unmarshal(respLine, &resp); err != nil {
		return ipcResponse{}, true, err
	}
	if !resp.OK {
		return resp, true, errors.New(resp.Error)
	}
	return resp, true, nil
}

// DaemonSend sends a one-way message through a running daemon. handled is false
// when no daemon is listening, so the caller should send directly instead.
func DaemonSend(to, text string) (handled bool, msgID, chatID int64, err error) {
	resp, handled, err := roundTrip(ipcRequest{Op: "send", To: to, Text: text}, time.Now().Add(30*time.Second))
	return handled, resp.MessageID, resp.ChatID, err
}

// DaemonAsk sends a question through a running daemon and blocks until the user
// replies (no timeout, matching standalone `tg ask`). handled is false when no
// daemon is listening.
func DaemonAsk(to, text string) (handled bool, reply string, err error) {
	resp, handled, err := roundTrip(ipcRequest{Op: "ask", To: to, Text: text}, time.Time{})
	return handled, resp.Reply, err
}

// DaemonChatWait routes a `tg chat` turn through the daemon: it optionally sends
// text (empty = just listen) and waits for the user's next reply, up to timeout
// (0 = no limit). handled is false when no daemon is listening.
func DaemonChatWait(to, text string, timeout time.Duration) (handled bool, reply string, err error) {
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}
	resp, handled, err := roundTrip(ipcRequest{Op: "ask", To: to, Text: text}, deadline)
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return handled, "", fmt.Errorf("timed out after %s waiting for a reply", timeout)
	}
	return handled, resp.Reply, err
}

// DaemonSendFile sends a local file through a running daemon and waits for the
// upload to finish. handled is false when no daemon is listening.
func DaemonSendFile(to, path, caption string) (handled bool, label string, chatID int64, err error) {
	resp, handled, err := roundTrip(ipcRequest{Op: "sendfile", To: to, Path: path, Text: caption}, time.Now().Add(31*time.Minute))
	return handled, resp.Label, resp.ChatID, err
}

// DaemonRunning reports whether a bridge daemon is listening on the socket.
func DaemonRunning() bool {
	conn, ok := dialDaemon()
	if ok {
		_ = conn.Close()
	}
	return ok
}
