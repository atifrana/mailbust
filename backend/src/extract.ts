/**
 * Verification code and link extraction.
 *
 * Shared by the email handler, REST API, CLI, and MCP server. The approach:
 *
 *   codes - scan subject and body, generate numeric/alphanumeric candidates,
 *           drop spans that are clearly something else (years, prices, phone
 *           numbers, dates, order/reference numbers), then score what's left by
 *           how close it sits to a code-hint word. Subject matches get a bonus.
 *
 *   links - collect URLs from the text and from HTML anchors (using the anchor
 *           text as a strong signal), drop unsubscribe/tracking/social/legal
 *           links, then score by verification-intent keywords in the URL and
 *           its surrounding context.
 *
 * Both return null rather than guess when nothing scores above a confidence
 * threshold - a wrong code is worse than no code.
 */

// ---- codes -----------------------------------------------------------------

const CODE_HINT =
  /(verification|verify|one[-\s]?time|passcode|pass\s?code|security\s+code|access\s+code|confirmation|confirm|authenticate|authentication|two[-\s]?factor|2fa|otp|\bpin\b|\bcode\b|log[-\s]?in|sign[-\s]?in|enter\s+(?:the|this))/i;

const YEAR = /^(?:19|20)\d{2}$/;

/** Spans that look code-like but are something else - excluded from candidates. */
function excludedSpans(src: string): Array<[number, number]> {
  const spans: Array<[number, number]> = [];
  const patterns = [
    // Phone numbers: optional country code, area code, 3+4 digits.
    /(?:\+?\d{1,3}[-.\s])?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}/g,
    // Numeric dates: 2025-10-21, 10/21/2025, 21.10.25.
    /\b\d{1,4}[/.\-]\d{1,2}[/.\-]\d{1,4}\b/g,
    // Money: $1,234.56, £99, or a bare decimal amount like 1234.56.
    /(?:[$€£]\s?\d[\d,]*(?:\.\d+)?|\b\d[\d,]*\.\d{2}\b)/g,
    // Order / reference / tracking numbers, incl. "#12345".
    /(?:#\s?\d{3,}|\b(?:order|invoice|inv|receipt|statement|ticket|ref(?:erence)?|account|acct|tracking|po)\b\s*(?:number|no\.?|#|:)?\s*[A-Z]?\d{3,})/gi,
  ];
  for (const re of patterns) {
    for (const m of src.matchAll(re)) {
      const i = m.index ?? 0;
      spans.push([i, i + m[0].length]);
    }
  }
  return spans;
}

interface Candidate {
  value: string;
  start: number;
  end: number;
}

function inExcluded(spans: Array<[number, number]>, s: number, e: number): boolean {
  return spans.some(([xs, xe]) => s < xe && e > xs);
}

function overlaps(taken: Array<[number, number]>, s: number, e: number): boolean {
  return taken.some(([xs, xe]) => s < xe && e > xs);
}

/** Generate code candidates from one source string, highest-priority first. */
function candidates(src: string): Candidate[] {
  const excluded = excludedSpans(src);
  const taken: Array<[number, number]> = [];
  const out: Candidate[] = [];

  const add = (value: string, start: number, end: number) => {
    if (inExcluded(excluded, start, end) || overlaps(taken, start, end)) return;
    taken.push([start, end]);
    out.push({ value, start, end });
  };

  // Grouped digits shown with a separator: "123-456" / "1234 5678".
  for (const m of src.matchAll(/\b(\d{3,4})[-\s](\d{3,4})\b/g)) {
    const joined = m[1]! + m[2]!;
    if (joined.length >= 4 && joined.length <= 8) {
      add(joined, m.index ?? 0, (m.index ?? 0) + m[0].length);
    }
  }
  // Google-style "G-123456" → keep just the digits.
  for (const m of src.matchAll(/\bG-(\d{4,8})\b/g)) {
    add(m[1]!, m.index ?? 0, (m.index ?? 0) + m[0].length);
  }
  // Plain digit runs.
  for (const m of src.matchAll(/\b\d{4,8}\b/g)) {
    add(m[0], m.index ?? 0, (m.index ?? 0) + m[0].length);
  }
  // Uppercase alphanumeric codes that contain at least one digit and one letter.
  for (const m of src.matchAll(/\b[A-Z0-9]{4,8}\b/g)) {
    if (/\d/.test(m[0]) && /[A-Z]/.test(m[0])) {
      add(m[0], m.index ?? 0, (m.index ?? 0) + m[0].length);
    }
  }
  return out;
}

function scoreCandidate(src: string, c: Candidate, isSubject: boolean): number {
  const before = src.slice(Math.max(0, c.start - 30), c.start);
  const after = src.slice(c.end, c.end + 20);
  const hintBefore = CODE_HINT.test(before);
  const hintAfter = CODE_HINT.test(after);
  const hasLetter = /[A-Z]/.test(c.value);

  // Reject lookalikes with no supporting hint.
  if (YEAR.test(c.value) && !hintBefore && !hintAfter) return -1;
  if (hasLetter && !hintBefore && !hintAfter) return -1;

  let score = 0;
  if (hintBefore) score += 3;
  if (hintAfter) score += 2;
  if (isSubject) score += 2;
  score += c.value.length === 6 ? 2 : 1; // 6 digits is the classic OTP length
  return score;
}

/** Pick the most likely verification code from subject + body text. */
export function extractCode(subject: string, text: string): string | null {
  let best: string | null = null;
  let bestScore = 3; // accept only scores strictly above this
  let hasIntent = false;
  const numericCandidates = new Set<string>();

  for (const [src, isSubject] of [
    [subject, true],
    [text, false],
  ] as const) {
    if (!src) continue;
    if (CODE_HINT.test(src)) hasIntent = true;
    for (const c of candidates(src)) {
      const s = scoreCandidate(src, c, isSubject);
      if (s > bestScore) {
        bestScore = s;
        best = c.value;
      }
      if (!/[A-Z]/.test(c.value) && !YEAR.test(c.value)) {
        numericCandidates.add(c.value);
      }
    }
  }

  if (best) return best;

  // Fallback: the email clearly is a code email and there's exactly one
  // unambiguous number, but it sits far from the hint word. Take it.
  if (hasIntent && numericCandidates.size === 1) {
    return [...numericCandidates][0]!;
  }
  return null;
}

// ---- links -----------------------------------------------------------------

const LINK_HINT =
  /(verify|confirm|activate|magic|login|log-in|signin|sign-in|reset|token|validate|authenticate|set-?password|accept|invite|invitation|onboard)/i;

const LINK_DENY =
  /(unsubscribe|optout|opt-out|\/privacy|\/terms|\/legal|\/help|\/support|mailto:|tel:|\.(?:png|gif|jpe?g|css)(?:[?#]|$)|facebook\.com|twitter\.com|\/\/x\.com|instagram\.com|linkedin\.com|youtube\.com|tiktok\.com|list-manage\.com\/unsubscribe)/i;

const URL_RE = /https?:\/\/[^\s"'<>)\]]+/gi;
const ANCHOR_RE = /<a\s[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi;

function decodeEntities(s: string): string {
  return s
    .replace(/&amp;/g, "&")
    .replace(/&#x2F;/gi, "/")
    .replace(/&#47;/g, "/")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">");
}

function stripTrailingPunct(url: string): string {
  return url.replace(/[).,;:!?'"]+$/, "");
}

interface LinkCandidate {
  url: string;
  anchor: string;
  context: string;
}

/** Pick the most likely verification/magic/reset link. */
export function extractLink(text: string, html: string | null): string | null {
  const cands: LinkCandidate[] = [];

  if (text) {
    for (const m of text.matchAll(URL_RE)) {
      const i = m.index ?? 0;
      cands.push({
        url: stripTrailingPunct(decodeEntities(m[0])),
        anchor: "",
        context: text.slice(Math.max(0, i - 40), i + m[0].length + 40),
      });
    }
  }

  if (html) {
    for (const m of html.matchAll(ANCHOR_RE)) {
      cands.push({
        url: stripTrailingPunct(decodeEntities(m[1]!)),
        anchor: m[2]!.replace(/<[^>]+>/g, " ").trim(),
        context: m[2]!,
      });
    }
    // Also catch bare URLs in HTML that aren't wrapped in an anchor.
    const stripped = html.replace(ANCHOR_RE, " ");
    for (const m of stripped.matchAll(URL_RE)) {
      cands.push({
        url: stripTrailingPunct(decodeEntities(m[0])),
        anchor: "",
        context: "",
      });
    }
  }

  let best: string | null = null;
  let bestScore = 2; // accept only scores strictly above this

  for (const c of cands) {
    if (LINK_DENY.test(c.url)) continue;
    let score = 0;
    if (LINK_HINT.test(c.anchor)) score += 3;
    if (LINK_HINT.test(c.url)) score += 3;
    if (LINK_HINT.test(c.context)) score += 1;
    if (score > bestScore) {
      bestScore = score;
      best = c.url;
    }
  }
  return best;
}
