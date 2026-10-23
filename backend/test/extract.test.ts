import { describe, it, expect } from "vitest";
import { extractCode, extractLink } from "../src/extract";

describe("extractCode", () => {
  it("pulls a code from the subject", () => {
    expect(extractCode("Your code is 482913", "")).toBe("482913");
  });

  it("pulls a hinted code from the body", () => {
    expect(
      extractCode("Verify your email", "Enter this code: 123456 to continue"),
    ).toBe("123456");
  });

  it("prefers a hinted code over an unrelated number", () => {
    const text = "Order #90210 confirmed. Your verification code is 55512.";
    expect(extractCode("", text)).toBe("55512");
  });

  it("ignores years", () => {
    expect(extractCode("Welcome", "Copyright 2025 Example Inc.")).toBeNull();
  });

  it("handles alphanumeric codes when labeled", () => {
    expect(extractCode("", "Your one-time code: A1B2C3")).toBe("A1B2C3");
  });

  it("returns null when there is no code", () => {
    expect(extractCode("Newsletter", "Thanks for reading our weekly update.")).toBeNull();
  });
});

describe("extractLink", () => {
  it("picks a verify link", () => {
    const text = "Confirm here: https://example.com/verify?token=abc123";
    expect(extractLink(text, null)).toBe("https://example.com/verify?token=abc123");
  });

  it("excludes unsubscribe links", () => {
    const text = "Unsubscribe: https://example.com/unsubscribe?u=9";
    expect(extractLink(text, null)).toBeNull();
  });

  it("prefers a reset link over a plain homepage", () => {
    const text =
      "Visit https://example.com/home or reset at https://example.com/reset?t=xyz";
    expect(extractLink(text, null)).toBe("https://example.com/reset?t=xyz");
  });

  it("strips trailing punctuation", () => {
    const text = "Activate your account at https://example.com/activate?id=7.";
    expect(extractLink(text, null)).toBe("https://example.com/activate?id=7");
  });
});
