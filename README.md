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

`messages`, `queue`, and `songs` each work two ways: run bare for an
interactive arrow-key menu (`tapnoted songs`), or give the subcommand
directly for a fast, scriptable path (`tapnoted songs add <url>`). In any
menu, **Esc** backs out one level (e.g. out of a delete confirmation back to
the main menu) instead of exiting the whole command; **Ctrl+C** still quits
immediately from anywhere. Network requests time out after 10 seconds, with
one automatic retry on a connection blip before giving up.

`replies` is **view-once**: listing them clears them from the server as part
of the same request (the same idea as how a tap already destructively pops
the queue) — so there's no separate `list`/`clear` subcommand, no menu, just
`tapnoted replies`. Use `overview` for a safe peek at how many are waiting
without consuming them.

| Command                                    | Purpose                                                                 |
| ------------------------------------------- | ------------------------------------------------------------------------ |
| `config set --url <url> --secret <secret>` | Save connection details                                                 |
| `config show`                              | Show the saved config (secret masked)                                   |
| `overview`                                 | Quick summary: queue count, pool size, song count, replies waiting (non-destructive) |
| `messages` / `messages add "<msg>"`        | Manage the random-pick pool — add, `list`, `delete [<index>]`, `reset`  |
| `queue` / `queue add ["<msg>"]`            | Manage the queue — add (interactive picker if no text given), `status`, `cancel [<index>]` |
| `songs` / `songs add <url> ["title"]`      | Manage the curated song pool — add, `list`, `delete [<index>]`, `reset` |
| `replies`                                  | Show her replies (message she replied to, her reply, and when) — clears them once shown |

## A note on how this was built

The idea and design behind this project are mine — a companion CLI to
manage [tapnote](https://github.com/Iwe-Coumou/tapnote) from the terminal,
built as an extension of that same learning project. I directed the scope
and design decisions (what it needed to do, how secrets should be handled,
how it should feel to use) and reviewed and tested every piece with AI
assistance (Claude) as it was built. AI acted as a pair programmer here, not
an autopilot.
