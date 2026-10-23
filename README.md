# MailBust

A disposable email inbox that lives in your terminal. Run one command and you get a
throwaway address that shows new mail the moment it arrives, pulls the verification
code out for you, and deletes itself after 15 minutes.

```
mailbust                        # a random private inbox, e.g. k7f2p9@mailbust.com
mailbust testing                # opens testing@mailbust.com (public, shared by name)
mailbust testing --code         # wait for the next code, print it, exit
```

Every inbox is an address on mailbust.com. Open a name and you get
`<name>@mailbust.com`; open nothing and you get a random `@mailbust.com` address.

## What it's good for

Signing up for things, testing the email your own app sends, or grabbing a one-time
code in CI, without handing over your real inbox or clicking around a webmail page.

- New mail shows up instantly over a WebSocket. No refreshing.
- The verification code (or login link) is pulled out and shown next to each message,
  so you rarely need to open an email.
- It works in scripts too: `--code`, `--link`, `--json`, and real exit codes.
- Random inboxes are private, protected by a token saved on your machine. Named
  inboxes are public: anyone who opens the same name sees the same mail.
- Everything is deleted when the inbox expires.

## Install

macOS / Linux:

```bash
curl -sSL https://raw.githubusercontent.com/atifrana/mailbust/main/docs/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/atifrana/mailbust/main/docs/install.ps1 | iex
```

With Go:

```bash
go install github.com/atifrana/mailbust/cli@latest
```

Or download a binary from the [releases page](https://github.com/atifrana/mailbust/releases).
Homebrew and Scoop are on the way.

## Using it

### Interactive inbox

```bash
mailbust                 # random private inbox (token saved in your OS keychain)
mailbust testing         # public inbox, shared by anyone who opens the same name
mailbust testing -t 1h   # give it a 1-hour lifetime (24h max on the free tier)
```

Keys inside the inbox:

| Key | Action | Key | Action |
| --- | ------ | --- | ------ |
| `↑` `↓` | move | `a` | copy the address |
| `enter` | open a message | `o` | open the link in a browser |
| `esc` | back to the list | `d` | delete |
| `c` | copy the code | `e` | add more time |
| `l` | copy the link | `t` | change the color theme |
| `r` | refresh | `q` | quit |

### Scripting and CI

The non-interactive mode prints only what you asked for, so it drops into a shell:

```bash
mailbust --new --json                  # {address, token, expires_at}
mailbust testing --code --timeout 90   # wait for a code, print it
mailbust testing --list --json         # current messages as JSON
```

Exit codes: `0` ok, `1` error, `2` timeout, `3` bad input. There's a full guide and a
ready-to-copy GitHub Actions workflow in [docs/scripting.md](docs/scripting.md).

### For AI agents (MCP)

`mailbust mcp` runs a Model Context Protocol server so an agent can create inboxes and
read codes on its own:

```bash
claude mcp add mailbust -- mailbust mcp
```

The tools are `create_mailbox`, `wait_for_code`, `wait_for_link`, `list_messages`, and
`read_message`. Setup for Claude Code, Claude Desktop, and Cursor is in
[docs/mcp.md](docs/mcp.md).

### Configuration

| Variable | Default | What it does |
| -------- | ------- | ------------ |
| `MAILBUST_API` | `https://api.mailbust.com` | backend base URL |
| `MAILBUST_DOMAIN` | `mailbust.com` | domain for bare names |
| `MAILBUST_TOKEN` | - | token for a private inbox, for CI where there's no keychain |

`NO_COLOR` is respected.

## How it works

```
Sender --email--> Cloudflare Email Routing (catch-all *@mailbust.com)
                     |
              Cloudflare Worker (TypeScript)
              |- email(): parse, extract code/link, store
              |- REST API (/v1/...) + long-poll wait
              |- WebSocket (live inbox)
                     |
         Durable Object, one per inbox
              |- SQLite storage for messages
              |- holds the WebSockets, pushes new mail
              |- deletes everything when the inbox expires
                     |
                R2 bucket for attachments
```

One Durable Object per inbox gives us storage, the live WebSocket push, and automatic
expiry in one place, with no separate database or socket server to run.

The Go CLI (built with Bubble Tea) and the Cloudflare Workers backend live in this
repo under [`/cli`](cli) and [`/backend`](backend).

## Development

```bash
# backend
cd backend && npm install && npm test && npm run dev

# cli
cd cli && go test ./... && go run . testing
```

MailBust only receives mail. It can't send, on purpose, so a throwaway domain never
turns into a spam source.

## License

[MIT](LICENSE)
