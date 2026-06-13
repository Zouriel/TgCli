# Agent bridge (`tg daemon`)

`tg daemon` turns the logged-in Telegram account into a two-way bridge: allow-listed
users message the account and drive **Claude Code** sessions on this machine (pick a
project, resume a past session with a summary, chat back and forth), while Claude's own
`tg send` / `tg ask` notifications keep working through the same account.

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

**`agent-allowlist.json`** — who may use the bridge, and at what role:
```json
{
  "@you":      { "role": "full",  "locations": ["*"] },
  "@teammate": { "role": "read",  "locations": ["Docssite"] }
}
```

**`agent-locations.json`** — the projects, with an optional per-location ceiling:
```json
{
  "App":  "/home/you/app",
  "Prod": { "path": "/home/you/prod", "max_role": "read" }
}
```

### Roles

| Role | What Claude may do | `claude` permission mode |
|---|---|---|
| `read` | inspect / plan only, never acts | `plan` |
| `confirm` | plans first, acts only after you reply `yes` in Telegram | `plan` → execute |
| `edit` | read/write/run, auto-approved | `acceptEdits` |
| `full` | anything, unattended | `--dangerously-skip-permissions` |

A location's `max_role` caps everyone there (effective role = the more restrictive of the
user's role and the location cap).

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
