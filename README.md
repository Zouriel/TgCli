# TgCli

A minimal Telegram CLI built on [TDLib](https://github.com/tdlib/td). Send messages, follow chats, and ask questions — all from your terminal.

Works on **Windows** and **Linux**.

---

## Install

### Pre-built binary (Windows)

Download the latest `TgCli-win64.zip` from [Releases](../../releases), extract it, and add the folder to your PATH.

### Build from source

```sh
git clone https://github.com/Zouriel/TgCli
cd TgCli
go build -o tg .
```

#### Linux — TDLib dependency

```sh
# Debian / Ubuntu
sudo apt install libtdjson-dev

# Arch
sudo pacman -S tdlib
```

Then place `libtdjson.so` next to the `tg` binary, or ensure it's in your library path.

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
| `tg ask @username <message>` | Send a message and **wait for their reply** |
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

Incoming media is **downloaded automatically** while you tail a chat, into a per-chat folder:

```
~/Downloads/telegramcli/<chat name>/
```

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
