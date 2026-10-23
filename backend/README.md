# MailBust Backend

Cloudflare Worker + Durable Object that receives disposable-mailbox email, extracts
verification codes/links, stores messages per mailbox, and expires them automatically.

## Develop

```bash
npm install
npm run typecheck   # tsc --noEmit
npm test            # vitest unit tests
npm run dev         # local wrangler dev server
```

## Deploy (one-time setup on your Cloudflare account)

You need a Cloudflare account with the `mailbust.com` domain on Cloudflare DNS.

1. **Log in**

   ```bash
   npx wrangler login
   ```

2. **Create the R2 bucket** (enable R2 in the dashboard first if prompted)

   ```bash
   npx wrangler r2 bucket create mailbust-attachments
   ```

3. **Deploy the Worker**

   ```bash
   npm run deploy
   ```

4. **Point email at the Worker.** In the Cloudflare dashboard:
   `mailbust.com` → **Email** → **Email Routing** → enable it, then add a
   **catch-all** rule with action **Send to a Worker → `mailbust`**.

## Smoke test (the M1 "done" check)

Send an email from Gmail to `test1@mailbust.com`, then:

```bash
# Read what arrived (named mailboxes are public - no token needed)
curl https://mailbust.<your-subdomain>.workers.dev/v1/mailboxes/test1@mailbust.com/messages
```

You should see the message with its extracted `code` / `link`.

## REST API (M1)

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/v1/mailboxes` | body `{name?, ttl?}` → `{address, token?, public, ...}` |
| GET | `/v1/mailboxes/:address` | mailbox info |
| GET | `/v1/mailboxes/:address/messages` | inbox list (incl. code/link) |
| GET | `/v1/mailboxes/:address/messages/:id` | full message |
| DELETE | `/v1/mailboxes/:address/messages/:id` | delete one |
| GET | `/v1/mailboxes/:address/messages/:id/attachments/:n` | download attachment |

Also: `GET /v1/mailboxes/:address/wait?for=code|link|any&timeout=`,
`GET /v1/mailboxes/:address/ws` (WebSocket), `POST /v1/mailboxes/:address/extend`,
and `GET /v1/health` (liveness probe).

Private mailboxes require `Authorization: Bearer <token>`.

## Operations

- **Rate limits** (per IP): mailbox creation 20/60s (`RL_CREATE`), wait calls
  120/60s (`RL_WAIT`). Over the limit returns `429`.
- **Size/volume limits**: messages over 10 MB are rejected; max 100 messages per
  mailbox (oldest dropped).
- **Uptime monitoring**: point an external monitor (UptimeRobot, Cloudflare Health
  Checks, etc.) at `https://api.mailbust.com/v1/health`.
- **Error logging**: Workers observability is enabled; `fetch`/`email` wrap errors
  with `console.error`. Tail live logs with `npx wrangler tail`.
- **Abuse contact**: to make `abuse@mailbust.com` deliver to a real inbox, add an
  Email Routing rule for that exact address (→ your inbox) in the Cloudflare
  dashboard; it takes precedence over the catch-all Worker route.
