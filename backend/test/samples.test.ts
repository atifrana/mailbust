import { describe, it, expect } from "vitest";
import { extractCode, extractLink } from "../src/extract";
import { SAMPLE_EMAILS } from "./fixtures/emails";

describe("sample email suite", () => {
  it("has at least 30 samples", () => {
    expect(SAMPLE_EMAILS.length).toBeGreaterThanOrEqual(30);
  });

  for (const s of SAMPLE_EMAILS) {
    it(`code: ${s.name}`, () => {
      expect(extractCode(s.subject, s.text)).toBe(s.code);
    });

    it(`link: ${s.name}`, () => {
      expect(extractLink(s.text, s.html ?? null)).toBe(s.link);
    });
  }
});
