# tgstream

[![Release](https://img.shields.io/github/v/release/Soupboy006/tgstream)](https://github.com/Soupboy006/tgstream/releases)
[![License](https://img.shields.io/github/license/Soupboy006/tgstream)](LICENSE)
[![Go](https://img.shields.io/github/go-mod-go-version/Soupboy006/tgstream)](go.mod)

Stream Telegram videos directly to your device — no downloading, no waiting for the full file.

`tgstream` connects to your own Telegram account and streams video straight from Telegram's servers to your local video player, using range requests so you can seek/scrub just like a normal video file.

## Features

- Stream video from private channels, public channels, groups, and DMs you have access to
- No files saved to disk — video is fetched chunk by chunk and passed through
- Full seek/scrub support
- Original audio tracks and subtitles (the container is passed through untouched)
- Runs locally: Telegram traffic uses your account, and the stream server only listens on `127.0.0.1` with a random URL token

## Requirements

- A Telegram API ID and API hash from [my.telegram.org](https://my.telegram.org) → API development tools
- Go 1.26+ to build from source (or use a release binary)

## Install

Download the binary for your platform from the [Releases page](https://github.com/Soupboy006/tgstream/releases).

**Linux / macOS**

```bash
chmod +x tgstream-<platform>
sudo mv tgstream-<platform> /usr/local/bin/tgstream
```

## Configure

`tgstream` does not ship API credentials. Export your own before running:

```bash
export TGSTREAM_API_ID=12345
export TGSTREAM_API_HASH=your_api_hash
```

Use the same values every run so the saved session keeps working. Never commit these values.

## Usage

From the repo:

```bash
go build -o tgstream .
./tgstream play https://t.me/somechannel/123
```

Supported link formats:

```text
https://t.me/c/123456789/42        private channel
https://t.me/somechannel/42        public channel, group, or DM
tg://resolve?domain=somechannel&post=42
```

The command prints a URL like `http://127.0.0.1:41234/play/<token>`; open it in VLC, mpv, or any player that supports HTTP range requests.

On first run you will be asked for your phone number, the login code, and your 2FA password (if set). The session is saved to `~/.tgstream/session.json` — treat it like a password and keep the directory private.
