# tgstream

[![Release](https://img.shields.io/github/v/release/Soupboy006/tgstream)](https://github.com/Soupboy006/tgstream/releases)
[![License](https://img.shields.io/github/license/Soupboy006/tgstream)](LICENSE)
[![Go](https://img.shields.io/github/go-mod-go-version/Soupboy006/tgstream)](go.mod)

Stream Telegram videos directly to your device — no downloading, no waiting for the full file.

`tgstream` connects to your own Telegram account and streams video straight from Telegram's servers to your local video player, using range requests so you can seek/scrub just like a normal video file.

## Features

- Stream video from any chat you have access to — private channels, public channels, and DMs
- No files saved to disk — nothing downloaded, just streamed
- Full seek/scrub support
- Multi-audio track and subtitle support (whatever's in the original file)
- Runs 100% locally — nothing leaves your device

## Install

Download the binary for your platform from the [Releases page](https://github.com/Soupboy006/tgstream/releases).

**Linux / macOS**

```bash
chmod +x tgstream-<platform>
sudo mv tgstream-<platform> /usr/local/bin/tgstream
