/** Address parsing, validation, and normalization rules for mailbox names. */

const LOCAL_RE = /^[a-z0-9]([a-z0-9.-]{1,38})[a-z0-9]$/;

/** Names we never hand out as public mailboxes. */
export const RESERVED_NAMES = new Set([
  "admin",
  "postmaster",
  "abuse",
  "support",
  "root",
  "noreply",
  "no-reply",
  "webmaster",
  "security",
  "hostmaster",
]);

const ALPHABET = "abcdefghijkmnpqrstuvwxyz23456789"; // no ambiguous chars (0/o/1/l)

/** Generate a random, unambiguous local part for anonymous mailboxes. */
export function randomLocalPart(len = 10): string {
  const bytes = crypto.getRandomValues(new Uint8Array(len));
  let out = "";
  for (const b of bytes) out += ALPHABET[b % ALPHABET.length];
  return out;
}

/** Whether a local part is a well-formed, non-reserved mailbox name. */
export function isAllowedLocal(local: string): boolean {
  return LOCAL_RE.test(local) && !RESERVED_NAMES.has(local);
}

export interface ParsedName {
  local: string;
  address: string;
}

/**
 * Normalize a user-supplied name or full address into a canonical address.
 * Accepts "testing" or "testing@domain"; the domain is always forced to ours.
 */
export function normalizeName(
  input: string,
  domain: string,
): ParsedName | { error: string } {
  const raw = input.trim().toLowerCase();
  const local = raw.includes("@") ? raw.slice(0, raw.indexOf("@")) : raw;

  if (local.length < 3 || local.length > 40) {
    return { error: "name must be 3-40 characters" };
  }
  if (!LOCAL_RE.test(local)) {
    return {
      error: "name may contain only lowercase letters, digits, dots, hyphens",
    };
  }
  if (RESERVED_NAMES.has(local)) {
    return { error: "name is reserved" };
  }
  return { local, address: `${local}@${domain}` };
}

/** Extract and normalize the local part from an inbound envelope recipient. */
export function recipientToAddress(
  recipient: string,
  domain: string,
): string | null {
  const at = recipient.lastIndexOf("@");
  if (at === -1) return null;
  const local = recipient.slice(0, at).trim().toLowerCase();
  if (!LOCAL_RE.test(local) || RESERVED_NAMES.has(local)) return null;
  return `${local}@${domain}`;
}
