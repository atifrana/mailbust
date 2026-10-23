# MailBust MCP server

`mailbust mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io)
server over stdio, so AI agents can create disposable mailboxes and read
verification codes while they work (e.g. to complete a signup flow end-to-end).

## Tools

| Tool | Arguments | Returns |
| ---- | --------- | ------- |
| `create_mailbox` | `name?`, `ttl?` | address + expiry (omit `name` for a random private mailbox) |
| `wait_for_code` | `address`, `timeout?` | the next verification code + sender/subject |
| `wait_for_link` | `address`, `timeout?` | the next verify/magic link + sender/subject |
| `list_messages` | `address` | current messages (sender, subject, code/link) |
| `read_message` | `address`, `message_id` | the plain-text body of one message |

Private mailboxes created via `create_mailbox` stay readable for the rest of the
session (the server remembers their tokens in memory). You can also set
`MAILBUST_TOKEN` in the server's `env` to read a specific private mailbox.

## Setup

**Claude Code**

```bash
claude mcp add mailbust -- mailbust mcp
```

**Claude Desktop** - add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mailbust": { "command": "mailbust", "args": ["mcp"] }
  }
}
```

**Cursor** - add to `.cursor/mcp.json` (project) or `~/.cursor/mcp.json` (global):

```json
{
  "mcpServers": {
    "mailbust": { "command": "mailbust", "args": ["mcp"] }
  }
}
```

If `mailbust` isn't on your `PATH`, use its absolute path as `command`. To point at
a non-default backend, add an `env` block with `MAILBUST_API` / `MAILBUST_DOMAIN`.

## Try it

Ask the agent: *"Create a disposable mailbox, tell me the address, then wait for a
verification code."* It will call `create_mailbox` then `wait_for_code`.
