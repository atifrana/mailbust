import { DurableObject } from "cloudflare:workers";
import type {
  Env,
  IncomingMessage,
  MailboxInfo,
  MessageSummary,
  StoredMessage,
  AttachmentMeta,
} from "./types";
import { json, error } from "./http";

const SCHEMA = `
CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS messages (
  id          TEXT PRIMARY KEY,
  from_addr   TEXT NOT NULL,
  from_name   TEXT,
  subject     TEXT NOT NULL,
  text_body   TEXT NOT NULL,
  html_body   TEXT,
  code        TEXT,
  link        TEXT,
  received_at INTEGER NOT NULL,
  size        INTEGER NOT NULL,
  attachments TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_received ON messages(received_at);
`;

interface Meta {
  address: string;
  public: boolean;
  token: string | null;
  createdAt: number;
  expiresAt: number;
}

/** One Durable Object instance per mailbox address. */
type WaitKind = "code" | "link" | "any";

interface Waiter {
  kind: WaitKind;
  resolve: (m: MessageSummary | null) => void;
  timer: ReturnType<typeof setTimeout>;
}

export class MailboxDO extends DurableObject<Env> {
  private sql: SqlStorage;
  private waiters = new Set<Waiter>();

  constructor(ctx: DurableObjectState, env: Env) {
    super(ctx, env);
    this.sql = ctx.storage.sql;
    this.sql.exec(SCHEMA);
  }

  override async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);
    const parts = url.pathname.split("/").filter(Boolean); // e.g. ["messages", ":id"]

    // Internal control-plane endpoints (called only by the Worker).
    if (parts[0] === "_init") return this.init(request);
    if (parts[0] === "_ingest") return this.ingest(request);

    // Named mailboxes open on demand: if this address has never been created,
    // provision it as a public mailbox (random/private ones always exist first,
    // created via POST with a token, so this only ever opens public names).
    let meta = this.meta();
    if (!meta) {
      const addr = request.headers.get("x-mb-address");
      if (!addr) return error(404, "mailbox_not_found", "mailbox does not exist");
      meta = await this.provisionPublic(addr, Number(request.headers.get("x-mb-ttl")) || 900);
    }

    const authErr = this.checkAuth(meta, request);
    if (authErr) return authErr;

    if (parts.length === 0) return json(this.info(meta));

    if (parts[0] === "extend") return this.extend(meta, request);

    if (parts[0] === "wait") return this.wait(url);

    if (parts[0] === "ws") return this.handleWebSocket(request);

    if (parts[0] === "messages") {
      if (parts.length === 1) return json({ messages: this.listMessages() });
      const id = parts[1]!;
      if (parts[2] === "attachments") {
        return this.getAttachment(id, Number(parts[3]));
      }
      if (request.method === "DELETE") return this.deleteMessage(id);
      return this.getMessage(id);
    }

    return error(404, "not_found", "unknown route");
  }

  // ---- control plane -------------------------------------------------------

  private async init(request: Request): Promise<Response> {
    const body = (await request.json()) as {
      address: string;
      public: boolean;
      token: string | null;
      ttl: number;
    };

    const existing = this.meta();
    if (existing) return json(this.info(existing)); // already created; stay idempotent

    const now = Date.now();
    const expiresAt = now + body.ttl * 1000;
    const meta: Meta = {
      address: body.address,
      public: body.public,
      token: body.token,
      createdAt: now,
      expiresAt,
    };
    this.writeMeta(meta);
    await this.ctx.storage.setAlarm(expiresAt);
    return json(this.info(meta));
  }

  private async ingest(request: Request): Promise<Response> {
    const { address, ttl, msg } = (await request.json()) as {
      address: string;
      ttl: number;
      msg: IncomingMessage;
    };

    // Named mail can arrive before anyone opens the mailbox: create on demand.
    if (!this.meta()) await this.provisionPublic(address, ttl);

    const id = crypto.randomUUID();
    const receivedAt = Date.now();
    this.sql.exec(
      `INSERT INTO messages
        (id, from_addr, from_name, subject, text_body, html_body, code, link, received_at, size, attachments)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      id,
      msg.from,
      msg.fromName,
      msg.subject,
      msg.text,
      msg.html,
      msg.code,
      msg.link,
      receivedAt,
      msg.size,
      JSON.stringify(msg.attachments),
    );

    await this.enforceCap();

    this.notifyWaiters({
      id,
      from: msg.from,
      fromName: msg.fromName,
      subject: msg.subject,
      code: msg.code,
      link: msg.link,
      receivedAt,
      hasAttachments: msg.attachments.length > 0,
    });

    return json({ id });
  }

  private async provisionPublic(address: string, ttl: number): Promise<Meta> {
    const now = Date.now();
    const meta: Meta = {
      address,
      public: true,
      token: null,
      createdAt: now,
      expiresAt: now + ttl * 1000,
    };
    this.writeMeta(meta);
    await this.ctx.storage.setAlarm(meta.expiresAt);
    return meta;
  }

  private async extend(meta: Meta, request: Request): Promise<Response> {
    if (request.method !== "POST") return error(405, "method", "use POST");
    const body = (await request.json().catch(() => ({}))) as { ttl?: number };

    const addSeconds =
      body.ttl && Number.isFinite(body.ttl) && body.ttl > 0
        ? Math.floor(body.ttl)
        : Number(this.env.DEFAULT_TTL_SECONDS);
    const maxExpiry = Date.now() + Number(this.env.MAX_TTL_SECONDS) * 1000;
    const expiresAt = Math.min(meta.expiresAt + addSeconds * 1000, maxExpiry);

    meta.expiresAt = expiresAt;
    this.writeMeta(meta);
    await this.ctx.storage.setAlarm(expiresAt);
    return json(this.info(meta));
  }

  override async alarm(): Promise<void> {
    // Mailbox expired: purge attachments from R2, then wipe all storage.
    const keys = this.sql
      .exec<{ attachments: string }>("SELECT attachments FROM messages")
      .toArray()
      .flatMap((r) => (JSON.parse(r.attachments) as AttachmentMeta[]).map((a) => a.key));
    await this.deleteAttachments(keys);
    await this.ctx.storage.deleteAll();
  }

  // ---- queries -------------------------------------------------------------

  private listMessages(): MessageSummary[] {
    return this.sql
      .exec<{
        id: string;
        from_addr: string;
        from_name: string | null;
        subject: string;
        code: string | null;
        link: string | null;
        received_at: number;
        attachments: string;
      }>(
        `SELECT id, from_addr, from_name, subject, code, link, received_at, attachments
         FROM messages ORDER BY received_at DESC`,
      )
      .toArray()
      .map((r) => ({
        id: r.id,
        from: r.from_addr,
        fromName: r.from_name,
        subject: r.subject,
        code: r.code,
        link: r.link,
        receivedAt: r.received_at,
        hasAttachments: (JSON.parse(r.attachments) as AttachmentMeta[]).length > 0,
      }));
  }

  private getMessage(id: string): Response {
    const row = this.sql
      .exec<MessageRow>("SELECT * FROM messages WHERE id = ?", id)
      .toArray()[0];
    if (!row) return error(404, "message_not_found", "no such message");
    return json(rowToMessage(row));
  }

  private deleteMessage(id: string): Response {
    const row = this.sql
      .exec<{ attachments: string }>(
        "SELECT attachments FROM messages WHERE id = ?",
        id,
      )
      .toArray()[0];
    if (!row) return error(404, "message_not_found", "no such message");

    const keys = (JSON.parse(row.attachments) as AttachmentMeta[]).map((a) => a.key);
    this.sql.exec("DELETE FROM messages WHERE id = ?", id);
    this.ctx.waitUntil(this.deleteAttachments(keys));
    return json({ deleted: true });
  }

  private async getAttachment(id: string, n: number): Promise<Response> {
    const row = this.sql
      .exec<{ attachments: string }>(
        "SELECT attachments FROM messages WHERE id = ?",
        id,
      )
      .toArray()[0];
    if (!row) return error(404, "message_not_found", "no such message");

    const att = (JSON.parse(row.attachments) as AttachmentMeta[]).find((a) => a.n === n);
    if (!att) return error(404, "attachment_not_found", "no such attachment");

    const obj = await this.env.ATTACHMENTS.get(att.key);
    if (!obj) return error(410, "attachment_gone", "attachment no longer stored");

    return new Response(obj.body, {
      headers: {
        "content-type": att.contentType,
        "content-length": String(att.size),
        "content-disposition": `attachment; filename="${att.filename.replace(/"/g, "")}"`,
      },
    });
  }

  // ---- long-poll wait ------------------------------------------------------

  private async wait(url: URL): Promise<Response> {
    const kind = parseKind(url.searchParams.get("for"));
    const timeout = clamp(Number(url.searchParams.get("timeout")) || 30, 1, 120);

    const existing = this.findMatch(kind);
    if (existing) return json({ message: existing });

    const match = await new Promise<MessageSummary | null>((resolve) => {
      const waiter: Waiter = {
        kind,
        resolve,
        timer: setTimeout(() => {
          this.waiters.delete(waiter);
          resolve(null);
        }, timeout * 1000),
      };
      this.waiters.add(waiter);
    });

    if (!match) return new Response(null, { status: 204 });
    return json({ message: match });
  }

  private notifyWaiters(summary: MessageSummary): void {
    for (const w of this.waiters) {
      if (matchesKind(w.kind, summary)) {
        clearTimeout(w.timer);
        this.waiters.delete(w);
        w.resolve(summary);
      }
    }
    // Push to any live WebSocket subscribers.
    const frame = JSON.stringify({ type: "message", message: summary });
    for (const ws of this.ctx.getWebSockets()) {
      try {
        ws.send(frame);
      } catch {
        // socket already closing; ignore
      }
    }
  }

  private handleWebSocket(request: Request): Response {
    if (request.headers.get("upgrade") !== "websocket") {
      return error(426, "upgrade_required", "expected a websocket upgrade");
    }
    const pair = new WebSocketPair();
    // Hibernatable: the runtime holds the socket; no in-memory bookkeeping.
    this.ctx.acceptWebSocket(pair[1]!);
    return new Response(null, { status: 101, webSocket: pair[0]! });
  }

  /** App-level ping keepalive; reply so clients can detect a live link. */
  override async webSocketMessage(ws: WebSocket, message: string | ArrayBuffer): Promise<void> {
    if (message === "ping") ws.send("pong");
  }

  private findMatch(kind: WaitKind): MessageSummary | null {
    const cond =
      kind === "code" ? "WHERE code IS NOT NULL" : kind === "link" ? "WHERE link IS NOT NULL" : "";
    const row = this.sql
      .exec<{
        id: string;
        from_addr: string;
        from_name: string | null;
        subject: string;
        code: string | null;
        link: string | null;
        received_at: number;
        attachments: string;
      }>(
        `SELECT id, from_addr, from_name, subject, code, link, received_at, attachments
         FROM messages ${cond} ORDER BY received_at DESC LIMIT 1`,
      )
      .toArray()[0];
    if (!row) return null;
    return {
      id: row.id,
      from: row.from_addr,
      fromName: row.from_name,
      subject: row.subject,
      code: row.code,
      link: row.link,
      receivedAt: row.received_at,
      hasAttachments: (JSON.parse(row.attachments) as AttachmentMeta[]).length > 0,
    };
  }

  // ---- helpers -------------------------------------------------------------

  private meta(): Meta | null {
    const rows = this.sql
      .exec<{ key: string; value: string }>("SELECT key, value FROM meta")
      .toArray();
    if (rows.length === 0) return null;
    const m = Object.fromEntries(rows.map((r) => [r.key, r.value]));
    if (!m.address) return null;
    return {
      address: m.address,
      public: m.public === "1",
      token: m.token ?? null,
      createdAt: Number(m.createdAt),
      expiresAt: Number(m.expiresAt),
    };
  }

  private writeMeta(meta: Meta): void {
    const rows: [string, string][] = [
      ["address", meta.address],
      ["public", meta.public ? "1" : "0"],
      ["createdAt", String(meta.createdAt)],
      ["expiresAt", String(meta.expiresAt)],
    ];
    if (meta.token) rows.push(["token", meta.token]);
    for (const [key, value] of rows) {
      this.sql.exec(
        "INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
        key,
        value,
      );
    }
  }

  private info(meta: Meta): MailboxInfo {
    return {
      address: meta.address,
      public: meta.public,
      createdAt: meta.createdAt,
      expiresAt: meta.expiresAt,
    };
  }

  private checkAuth(meta: Meta, request: Request): Response | null {
    if (meta.public || !meta.token) return null;
    const header = request.headers.get("authorization") ?? "";
    const token = header.startsWith("Bearer ") ? header.slice(7) : "";
    if (!timingSafeEqual(token, meta.token)) {
      return error(401, "unauthorized", "valid bearer token required");
    }
    return null;
  }

  private async enforceCap(): Promise<void> {
    const max = Number(this.env.MAX_MESSAGES);
    const stale = this.sql
      .exec<{ id: string; attachments: string }>(
        `SELECT id, attachments FROM messages
         ORDER BY received_at DESC LIMIT -1 OFFSET ?`,
        max,
      )
      .toArray();
    if (stale.length === 0) return;

    const keys = stale.flatMap(
      (r) => (JSON.parse(r.attachments) as AttachmentMeta[]).map((a) => a.key),
    );
    for (const r of stale) this.sql.exec("DELETE FROM messages WHERE id = ?", r.id);
    await this.deleteAttachments(keys);
  }

  private async deleteAttachments(keys: string[]): Promise<void> {
    if (keys.length > 0) await this.env.ATTACHMENTS.delete(keys);
  }
}

interface MessageRow extends Record<string, SqlStorageValue> {
  id: string;
  from_addr: string;
  from_name: string | null;
  subject: string;
  text_body: string;
  html_body: string | null;
  code: string | null;
  link: string | null;
  received_at: number;
  size: number;
  attachments: string;
}

function rowToMessage(r: MessageRow): StoredMessage {
  return {
    id: r.id,
    from: r.from_addr,
    fromName: r.from_name,
    subject: r.subject,
    text: r.text_body,
    html: r.html_body,
    code: r.code,
    link: r.link,
    receivedAt: r.received_at,
    size: r.size,
    attachments: JSON.parse(r.attachments) as AttachmentMeta[],
  };
}

function parseKind(raw: string | null): WaitKind {
  return raw === "code" || raw === "link" ? raw : "any";
}

function matchesKind(kind: WaitKind, m: MessageSummary): boolean {
  if (kind === "code") return m.code !== null;
  if (kind === "link") return m.link !== null;
  return true;
}

function clamp(n: number, min: number, max: number): number {
  return Math.min(Math.max(n, min), max);
}

/** Constant-time string comparison to avoid token-guessing timing leaks. */
function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
}
