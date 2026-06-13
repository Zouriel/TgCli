package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// TriageSettings configures the hourly importance triage of inbound messages
// from people who aren't on the allow-list.
type TriageSettings struct {
	Enabled      bool   `json:"enabled"`
	EveryMinutes int    `json:"every_minutes"` // how often to run (default 60)
	Dir          string `json:"dir"`           // working dir for the triage agent
	// The agent backend for triage is configured in agents.json (tasks.triage).
}

// Settings drives the auto-reply and triage features (agent-settings.json).
type Settings struct {
	MainUser         string         `json:"main_user"`          // @username to notify / send digests to
	AutoReplyEnabled bool           `json:"auto_reply_enabled"` // reply to non-allow-listed senders
	AutoReply        string         `json:"auto_reply"`         // the canned reply text
	Triage           TriageSettings `json:"triage"`
}

const settingsFile = "agent-settings.json"

// LoadOrSeedSettings reads agent-settings.json, creating a disabled placeholder
// on first run.
func LoadOrSeedSettings() (Settings, string, error) {
	dir, err := configDir()
	if err != nil {
		return Settings{}, "", err
	}
	path := filepath.Join(dir, settingsFile)

	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		seed := Settings{
			MainUser:         "@your_username",
			AutoReplyEnabled: false,
			AutoReply:        "Message received — the owner will be notified shortly.",
			Triage: TriageSettings{
				Enabled:      false,
				EveryMinutes: 60,
				Dir:          home,
			},
		}
		if err := writeJSON(path, seed); err != nil {
			return Settings{}, path, err
		}
		return seed, path, nil
	}
	if err != nil {
		return Settings{}, path, err
	}

	_ = os.Chmod(path, 0o600)

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, path, err
	}
	if settings.Triage.EveryMinutes <= 0 {
		settings.Triage.EveryMinutes = 60
	}
	if settings.Triage.Dir == "" {
		settings.Triage.Dir = home
	}
	return settings, path, nil
}
