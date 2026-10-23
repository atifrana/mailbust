import PostalMime from "postal-mime";
import type { Env, IncomingMessage, AttachmentMeta } from "./types";
import { extractCode, extractLink } from "./extract";
import { normalizeName, randomLocalPart, recipientToAddress, isAllowedLocal } from "./address";
import { json, error } from "./http";

export { MailboxDO } from "./mailbox";

export default {
  /** REST API. */
  async fetch(request: Request, env: Env): Promise<Response> {
    try {
      return await route(request, env);
    } catch (err) {
      console.error("unhandled error", {
        url: request.url,
        method: request.method,
        error: err instanceof Error ? err.stack ?? err.message : String(err),
      });
      return error(500, "internal_error", "something went wrong");
    }
  },

  /** Inbound email: parse, extract, store attachments, hand off to the DO. */
  async email(message: ForwardableEmailMessage, env: Env): Promise<void> {
    try {
      await handleEmail(message, env);
    } catch (err) {
      console.error("email handler error", {
        to: message.to,
        error: err instanceof Error ? err.stack ?? err.message : String(err),
      });
      throw err; // let the platform retry/queue as configured
    }
  },
} satisfies ExportedHandler<Env>;

/** Routes a REST request, applying per-IP rate limits to expensive endpoints. */
async function route(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const parts = url.pathname.split("/").filter(Boolean);

  if (parts[0] !== "v1") return error(404, "not_found", "unknown route");

  // GET /v1/health - liveness probe for uptime monitoring.
  if (parts[1] === "health") {
    return json({ status: "ok", time: new Date().toISOString() });
  }

  if (parts[1] !== "mailboxes") return error(404, "not_found", "unknown route");

  const ip = request.headers.get("cf-connecting-ip") ?? "unknown";

  // POST /v1/mailboxes  - create a mailbox (rate-limited per IP).
  if (parts.length === 2) {
    if (request.method !== "POST") return error(405, "method", "use POST");
    if (!(await env.RL_CREATE.limit({ key: ip })).success) {
      return error(429, "rate_limited", "too many mailboxes created; slow down");
    }
    return createMailbox(request, env);
  }

  // /v1/mailboxes/:address/...  - forward to the owning Durable Object.
  const address = decodeURIComponent(parts[2]!).toLowerCase();
  const at = address.lastIndexOf("@");
  const local = at >= 0 ? address.slice(0, at) : address;
  if (!isAllowedLocal(local)) {
    return error(400, "invalid_address", "invalid mailbox name");
  }
  const sub = "/" + parts.slice(3).join("/");

  // Long-poll waits are rate-limited per IP (they hold a connection open).
  if (parts[3] === "wait" && !(await env.RL_WAIT.limit({ key: ip })).success) {
    return error(429, "rate_limited", "too many wait requests; slow down");
  }

  const headers = new Headers(request.headers);
  headers.set("x-mb-address", address);
  headers.set("x-mb-ttl", env.DEFAULT_TTL_SECONDS);

  const stub = env.MAILBOX.get(env.MAILBOX.idFromName(address));
  const forwarded = new Request(new URL(sub, "https://do.internal") + url.search, {
    method: request.method,
    headers,
  });
  return stub.fetch(forwarded);
}

/** Parses and stores one inbound email. */
async function handleEmail(message: ForwardableEmailMessage, env: Env): Promise<void> {
    const address = recipientToAddress(message.to, env.MAILBUST_DOMAIN);
    if (!address) {
      message.setReject("No such mailbox");
      return;
    }

    const maxBytes = Number(env.MAX_MESSAGE_BYTES);
    if (message.rawSize > maxBytes) {
      message.setReject("Message too large");
      return;
    }

    const raw = new Uint8Array(await streamToBuffer(message.raw, maxBytes));
    const parsed = await PostalMime.parse(raw);

    const text = parsed.text ?? "";
    const html = parsed.html ?? null;
    const subject = parsed.subject ?? "";

    // Persist attachments to R2 before storing metadata.
    const id = crypto.randomUUID();
    const attachments: AttachmentMeta[] = [];
    let n = 0;
    for (const att of parsed.attachments ?? []) {
      const bytes = toBytes(att.content);
      const key = `${address}/${id}/${n}`;
      await env.ATTACHMENTS.put(key, bytes);
      attachments.push({
        n,
        filename: att.filename ?? `attachment-${n}`,
        contentType: att.mimeType ?? "application/octet-stream",
        size: bytes.byteLength,
        key,
      });
      n++;
    }

    const msg: IncomingMessage = {
      from: parsed.from?.address ?? message.from,
      fromName: parsed.from?.name || null,
      subject,
      text,
      html,
      code: extractCode(subject, text || htmlToText(html)),
      link: extractLink(text, html),
      size: message.rawSize,
      attachments,
    };

    const ttl = Number(env.DEFAULT_TTL_SECONDS);
    const stub = env.MAILBOX.get(env.MAILBOX.idFromName(address));
    await stub.fetch(
      new Request("https://do.internal/_ingest", {
        method: "POST",
        body: JSON.stringify({ address, ttl, msg }),
      }),
    );
}

async function createMailbox(request: Request, env: Env): Promise<Response> {
  const body = (await request.json().catch(() => ({}))) as {
    name?: string;
    ttl?: number;
  };

  const ttl = clampTtl(body.ttl, env);
  let address: string;
  let isPublic: boolean;
  let token: string | null;

  if (typeof body.name === "string" && body.name.trim()) {
    const parsed = normalizeName(body.name, env.MAILBUST_DOMAIN);
    if ("error" in parsed) return error(400, "invalid_name", parsed.error);
    address = parsed.address;
    isPublic = true; // named mailboxes are public in v1
    token = null;
  } else {
    address = `${randomLocalPart()}@${env.MAILBUST_DOMAIN}`;
    isPublic = false;
    token = randomLocalPart(32);
  }

  const stub = env.MAILBOX.get(env.MAILBOX.idFromName(address));
  const res = await stub.fetch(
    new Request("https://do.internal/_init", {
      method: "POST",
      body: JSON.stringify({ address, public: isPublic, token, ttl }),
    }),
  );
  const info = (await res.json()) as Record<string, unknown>;
  return json({ ...info, token: token ?? undefined }, 201);
}

function clampTtl(ttl: number | undefined, env: Env): number {
  const def = Number(env.DEFAULT_TTL_SECONDS);
  const max = Number(env.MAX_TTL_SECONDS);
  if (!ttl || !Number.isFinite(ttl) || ttl <= 0) return def;
  return Math.min(Math.floor(ttl), max);
}

async function streamToBuffer(
  stream: ReadableStream<Uint8Array>,
  limit: number,
): Promise<ArrayBuffer> {
  const chunks: Uint8Array[] = [];
  let total = 0;
  const reader = stream.getReader();
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.byteLength;
    if (total > limit) throw new Error("message exceeds size limit");
    chunks.push(value);
  }
  const out = new Uint8Array(total);
  let offset = 0;
  for (const c of chunks) {
    out.set(c, offset);
    offset += c.byteLength;
  }
  return out.buffer;
}

function toBytes(content: string | ArrayBuffer | Uint8Array): Uint8Array {
  if (content instanceof Uint8Array) return content;
  if (content instanceof ArrayBuffer) return new Uint8Array(content);
  return new TextEncoder().encode(content);
}

/** Minimal HTML→text fallback for code extraction when there is no text part. */
function htmlToText(html: string | null): string {
  if (!html) return "";
  return html
    .replace(/<style[\s\S]*?<\/style>/gi, " ")
    .replace(/<script[\s\S]*?<\/script>/gi, " ")
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}
