# tapnoted

A command-line tool for managing the messages behind
[tapnote](https://github.com/Iwe-Coumou/tapnote), an NFC tap-to-reveal
message gift. It talks to the deployed tapnote Cloudflare Worker over its
admin API — queue a specific message for the next tap, manage the random
message pool, all from the terminal.

## Prerequisites

- [Go](https://go.dev) (1.21+)
- A deployed tapnote Worker and its `ADMIN_SECRET` (see the
  [tapnote README](https://github.com/Iwe-Coumou/tapnote) for setting those up).

## Install

```
go install .
```

This builds the CLI and places it in your Go bin directory (`go env GOPATH`
+ `\bin`, e.g. `C:\Users\<you>\go\bin` on Windows). If that directory is on
your `PATH`, the `tapnoted` command is then available from anywhere.

Alternatively, `go build -o tapnoted.exe .` builds a binary into the current
folder without installing it globally.

## Setup

Point the CLI at your deployed Worker, once:

```
tapnoted config set --url https://tapnote.<your-subdomain>.workers.dev --secret <your-ADMIN_SECRET>
```

This is saved to a config file outside this project folder (in your OS user
config directory), so the secret never ends up in this repo. Check what's
currently saved (secret partially masked) with:

```
tapnoted config show
```

## Commands

| Command                          | Purpose                                                                 |
| --------------------------------- | ------------------------------------------------------------------------ |
| `config set --url <url> --secret <secret>` | Save connection details                                     |
| `config show`                     | Show the saved config (secret masked)                                   |
| `queue ["<message>"]`             | Add a message to the queue. No argument opens an interactive picker (existing pool message, or type a new one). Taps serve queued messages in order before falling back to random. |
| `status`                          | List what's currently queued, in order                                  |
| `cancel [<index>]`                | Clear the whole queue (asks to confirm), or remove just one item by index |
| `add "<message>"`                 | Add a message to the random pool                                        |
| `list`                            | List all messages in the pool                                           |
| `delete [<index>]`                | Remove a message from the pool. No argument opens an interactive picker with a confirmation step. |
| `reset`                           | Clear the entire pool (asks to confirm)                                 |

## A note on how this was built

The idea and design behind this project are mine — a companion CLI to
manage [tapnote](https://github.com/Iwe-Coumou/tapnote) from the terminal,
built as an extension of that same learning project. I directed the scope
and design decisions (what it needed to do, how secrets should be handled,
how it should feel to use) and reviewed and tested every piece with AI
assistance (Claude) as it was built. AI acted as a pair programmer here, not
an autopilot.
