# TgCli

A minimal Telegram CLI built on [TDLib](https://github.com/tdlib/td). Send messages, follow chats, and ask questions — all from your terminal.

Works on **Windows** and **Linux**.

---

## Install

### Pre-built binary (recommended)

Grab the bundle for your OS from [Releases](../../releases) — it ships with the
required TDLib library inside, so there's nothing else to install:

- **Windows** — `TgCli-win64.zip`: extract, then run `tg.exe` (or add the folder to your PATH).
- **Linux (x86-64)** — `TgCli-linux-x64.tar.gz`:
  ```sh
  tar xzf TgCli-linux-x64.tar.gz
  cd TgCli-linux-x64
  ./tg login
  ```
  `tg` loads the bundled `libtdjson.so` from its own folder, so just keep them together.

  > The bundled Linux library needs **glibc ≥ 2.38** and **OpenSSL 3** (Arch, Fedora 39+,
  > Ubuntu 24.04+). On older distros, use *Build from source* below.

### Build from source

```sh
git clone https://github.com/Zouriel/TgCli
cd TgCli
go build -o tg .
```

#### Linux — TDLib dependency

The repo bundles the Windows TDLib DLLs (`bin/`) but **not** the Linux `libtdjson.so`
(it's distro-specific). Install it from your package manager:

```sh
# Debian / Ubuntu
sudo apt install libtdjson-dev

# Arch
sudo pacman -S tdlib
```

Then place `libtdjson.so` next to the `tg` binary, or ensure it's in your library path.

### Packaging a release

`scripts/package.sh` builds the downloadable bundles into `dist/` (bundling a
`libtdjson.so` it finds locally, overridable with `LIBTDJSON=/path/to/libtdjson.so`).

---

## Setup

### 1. Get Telegram API credentials

Go to [my.telegram.org](https://my.telegram.org), create an app, and grab your **API ID** and **API Hash**.

### 2. Configure credentials

Copy `example.env` to `.env` and fill in your values:

```sh
cp example.env .env
```

```env
TG_API_ID   = "your_api_id"
TG_API_HASH = "your_api_hash"
```

> **Tip:** If you're distributing a pre-built binary, embed credentials at build time so end users don't need to supply them:
> ```sh
> go build -ldflags "-X tg/internal/config.BuildAPIID=123456 -X tg/internal/config.BuildAPIHash=abc123"
> ```

### 3. Login

```sh
tg login
```

Authenticate once with your phone number. The session is persisted locally — you won't need to log in again.

---

## Commands

| Command | Description |
|---|---|
| `tg login` | Sign in to Telegram |
| `tg logout [--hard]` | Sign out (`--hard` wipes the local session) |
| `tg auth` | Show the currently logged-in account |
| `tg send @username <message>` | Send a message to a user |
| `tg send-file <@username\|chat_id> <path> [caption]` | Send a file (photo/video/audio/document) |
| `tg download <@username\|chat_id>` | List recent received media and download a chosen one |
| `tg ask @username <message>` | Send a message and **wait for their reply** |
| `tg chat <@username\|chat_id> [message]` | One round-trip: send and/or wait for the next reply (scriptable) |
| `tg tail <@username\|chat_id>` | Follow a chat live; type to send, paste a path to send a file |
| `tg chats` | List recent chats |

### Examples

```sh
tg send @alice "deploy finished successfully"

tg send-file @alice ./report.pdf "here's the report"

tg ask @alice "should I use postgres or sqlite for this?" 
# blocks until @alice replies, prints their answer

tg tail @mygroup
```

### Media

Send a file from any chat by **pasting its path** while tailing, or with `tg send-file`.
The file type is chosen automatically from the extension — images go as photos, clips as
videos, audio as audio, everything else as a document.

Incoming media is **never downloaded automatically** — downloading an arbitrary file
just because it arrived is a security risk. Instead, fetch media deliberately with
`tg download`, which lists the most recent media in a chat (newest first) and lets you
pick one:

```sh
tg download @alice
# Recent media in "alice" (newest first):
#   [0] photo     sunset.jpg  — look at this  (1.2 MB)
#   [1] document  report.pdf  (340.0 KB)
# Enter number to download (Enter = newest [0], q to cancel):
```

Downloads land in a per-chat folder:

```
~/Downloads/telegramcli/<chat name>/
```

| Flag | Description |
|---|---|
| `-n, --limit <N>` | How many recent messages to scan for media (default 30) |
| `-p, --pick <i>` | Download index `i` non-interactively (`0` = newest) |
| `--json` | List available media as JSON and exit (no download) |

---

## Scriptable back-and-forth: `tg chat`

`tail` is the interactive REPL. `tg chat` is its non-interactive counterpart — each
invocation does **one round-trip and exits**, so it composes cleanly in scripts and
agent loops (no long-lived process holding the session lock).

```sh
# Send and wait for the reply (prints the reply to stdout)
reply=$(tg chat @you "deploy to prod now or wait?")

# Just wait for the next incoming message (let the other side start)
tg chat @you

# Catch up: snapshot the last 10 messages and exit
tg chat @you --read 10

# Structured output for programmatic use; bound the wait
tg chat @you "still there?" --json --timeout 2m
```

| Flag | Description |
|---|---|
| `-w, --wait` | Wait for the next reply after sending (default `true`) |
| `-t, --timeout <dur>` | Max time to wait, e.g. `90s`, `5m` (`0` = no limit) |
| `-r, --read <N>` | Snapshot mode: print the last N messages and exit |
| `--json` | Emit each message as a JSON line (`message_id`, `sender`, `kind`, `text`, `file`, …) |
| `--download` | Download media in the reply (off by default) |

Media in replies is **not** downloaded by default. Pass `--download` to fetch it (only
for trusted senders), or use `tg download` to pick a specific file. With `--download` +
`--json`, the saved path comes back in the `file` field.

---

## Using tg for programmatic notifications

`tg send` and `tg ask` are designed to be scripted. You can use them in CI, cron jobs, or AI agent workflows to stay in the loop when you're away from your desk.

**One-way notification:**
```sh
tg send @you "build passed ✅"
```

**Ask a question and use the answer:**
```sh
answer=$(tg ask @you "deploy to prod now or wait?")
echo "User said: $answer"
```

**In a script:**
```sh
#!/bin/bash
run_tests
if [ $? -eq 0 ]; then
  tg send @you "tests passed — deploying"
  deploy
  tg send @you "deployment done ✅"
else
  choice=$(tg ask @you "tests failed — retry or abort?")
  # act on $choice
fi
```

---

## Use with Claude / AI agents

There's a companion [**Claude Agent Skill**](https://github.com/Zouriel/tgcli-skill) that teaches
Claude Code (and any agent supporting the skills standard) how to drive `tg` — so it can notify you,
ask a question and wait for your reply, converse, and send/receive files on its own:

**→ [github.com/Zouriel/tgcli-skill](https://github.com/Zouriel/tgcli-skill)**

```sh
git clone https://github.com/Zouriel/tgcli-skill
cp -r tgcli-skill/skills/tg ~/.claude/skills/tg
```

Then tell the agent your Telegram `@username`, and it will reach you on Telegram when it finishes a
task or gets stuck.

---

## Environment variables

| Variable | Description |
|---|---|
| `TG_API_ID` | Telegram API ID |
| `TG_API_HASH` | Telegram API hash |
| `TDLIB_BIN` | Path to the directory containing TDLib binaries (optional) |

Variables can be set in a `.env` file in the current directory or exported in the shell.

---

## License

[MIT](LICENSE)
