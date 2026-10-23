/** Cloudflare Workers rate-limiting binding. */
export interface RateLimiter {
  limit(options: { key: string }): Promise<{ success: boolean }>;
}

/** Cloudflare bindings available to the Worker and Durable Object. */
export interface Env {
  MAILBOX: DurableObjectNamespace<import("./mailbox").MailboxDO>;
  ATTACHMENTS: R2Bucket;
  RL_CREATE: RateLimiter;
  RL_WAIT: RateLimiter;
  MAILBUST_DOMAIN: string;
  DEFAULT_TTL_SECONDS: string;
  MAX_TTL_SECONDS: string;
  MAX_MESSAGE_BYTES: string;
  MAX_MESSAGES: string;
}

/** Attachment metadata stored with a message (bytes live in R2). */
export interface AttachmentMeta {
  n: number;
  filename: string;
  contentType: string;
  size: number;
  key: string;
}

/** A fully stored message, as returned by the full-message endpoint. */
export interface StoredMessage {
  id: string;
  from: string;
  fromName: string | null;
  subject: string;
  text: string;
  html: string | null;
  code: string | null;
  link: string | null;
  receivedAt: number;
  size: number;
  attachments: AttachmentMeta[];
}

/** Compact row used by the inbox list. */
export interface MessageSummary {
  id: string;
  from: string;
  fromName: string | null;
  subject: string;
  code: string | null;
  link: string | null;
  receivedAt: number;
  hasAttachments: boolean;
}

/** Mailbox metadata returned by create/info endpoints. */
export interface MailboxInfo {
  address: string;
  public: boolean;
  createdAt: number;
  expiresAt: number;
}

/** Payload the email handler posts into the Durable Object. */
export interface IncomingMessage {
  from: string;
  fromName: string | null;
  subject: string;
  text: string;
  html: string | null;
  code: string | null;
  link: string | null;
  size: number;
  attachments: AttachmentMeta[];
}
