# Agent bridge (`tg init agent`)

`tg init agent` turns the logged-in Telegram account into a two-way bridge: allow-listed
users message the account and drive an **AI agent** (Claude Code or Codex) on this machine
(pick a project, resume a past session with a summary, chat back and forth), while your own
`tg send` / `tg ask` notifications keep working through the same account. It can also
auto-reply to strangers and DM you an hourly digest of important messages.

## How it runs

The daemon owns the single Telegram session and listens on a Unix socket
(`~/.config/tg/daemon.sock`, owner-only `0600`). `tg send`, `tg ask`, `tg chat`, and
`tg send-file` automatically route through that socket when the daemon is running, and
fall back to opening their own session when it isn't.

Commands that need their own interactive session — `tail`, `chats`, `download`, `auth`,
`login`, `logout` — can't share the daemon's session, so while it's running they exit
with a clear message instead of a lock error. Stop the daemon to use them:

```sh
systemctl --user stop tg-daemon
```

Run it as a service (stays up, restarts on crash, starts on boot):

```sh
cp scripts/tg-daemon.service ~/.config/systemd/user/   # edit WorkingDirectory if needed
systemctl --user enable --now tg-daemon
loginctl enable-linger "$USER"                          # start on boot without login
```

## Configuration

Two JSON files in the tg config dir (`~/.config/tg/`), both kept `0600`:

**`agent-allowlist.json`** — who may use the bridge, at what role, and (optionally) which
agent backend:
```json
{
  "@you":      { "role": "full",  "locations": ["*"],        "agent": "codex" },
  "@teammate": { "role": "read",  "locations": ["Docssite"]                    }
}
```
`agent` is `claude` (default) or `codex` — each user can drive either backend.

**`agent-locations.json`** — the projects, with an optional per-location ceiling:
```json
{
  "App":  "/home/you/app",
  "Prod": { "path": "/home/you/prod", "max_role": "read" }
}
```

### Roles

| Role | What the agent may do | Claude mode | Codex sandbox |
|---|---|---|---|
| `read` | inspect / plan only, never acts | `plan` | `read-only` |
| `confirm` | plans first, acts only after you reply `yes` in Telegram | `plan` → execute | `read-only` → execute |
| `edit` | read/write/run, auto-approved | `acceptEdits` | `workspace-write` |
| `full` | anything, unattended | `--dangerously-skip-permissions` | `--dangerously-bypass-approvals-and-sandbox` |

A location's `max_role` caps everyone there (effective role = the more restrictive of the
user's role and the location cap).

## Auto-reply & hourly triage (`agent-settings.json`)

Optional "secretary" features for messages from people **not** on the allow-list:

```json
{
  "main_user": "@you",
  "auto_reply_enabled": true,
  "auto_reply": "Message received — the owner will be notified shortly.",
  "triage": { "enabled": true, "every_minutes": 60, "dir": "/home/you", "agent": "claude" }
}
```

- **Auto-reply** — a non-allow-listed sender gets the canned `auto_reply` (at most once per
  hour each).
- **Triage** — every `every_minutes`, the buffered stranger messages are run through the
  `agent` (read-only) in `dir`; it decides which are important and DMs `main_user` a bullet
  digest. Nothing is sent if none are important.

These run even with an empty allow-list ("secretary-only" mode).

## ⚠️ Security model — read before adding anyone

- **Roles gate writes, not access.** Even `read` lets Claude open any file this Unix user
  can and send the contents back over Telegram. `edit`/`full` can touch the **whole
  filesystem** — `locations` choose where a session *starts*, they do **not** sandbox it.
- **Allow-listing someone ≈ giving them shell-level access** to this machine as your user.
  Only add people you'd trust with that. For untrusted users, isolate properly (a separate
  Unix user, a container, or a VM) — this tool does not sandbox.
- **`full` runs `--dangerously-skip-permissions`.** Whoever controls an allow-listed
  Telegram account can run arbitrary commands here. **Enable two-factor auth** on the
  Telegram account the daemon logs in as, and keep the allowlist as small as possible.
- The IPC socket is `0600` (owner-only), but any process running **as your user** can talk
  to it and send messages as the account — inherent to local IPC.
