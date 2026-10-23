import { describe, it, expect } from "vitest";
import { normalizeName, recipientToAddress, isAllowedLocal, RESERVED_NAMES } from "../src/address";

const DOMAIN = "mailbust.com";

describe("normalizeName", () => {
  it("accepts a bare name", () => {
    expect(normalizeName("testing", DOMAIN)).toEqual({
      local: "testing",
      address: "testing@mailbust.com",
    });
  });

  it("strips a supplied domain and lowercases", () => {
    expect(normalizeName("Testing@Other.com", DOMAIN)).toEqual({
      local: "testing",
      address: "testing@mailbust.com",
    });
  });

  it("rejects too-short names", () => {
    expect(normalizeName("ab", DOMAIN)).toHaveProperty("error");
  });

  it("rejects illegal characters", () => {
    expect(normalizeName("hi there", DOMAIN)).toHaveProperty("error");
  });

  it("rejects reserved names", () => {
    for (const name of RESERVED_NAMES) {
      expect(normalizeName(name, DOMAIN)).toHaveProperty("error");
    }
  });
});

describe("isAllowedLocal", () => {
  it("accepts valid names", () => {
    for (const n of ["testing", "a-b.c", "abc123", "x".repeat(40)]) {
      expect(isAllowedLocal(n)).toBe(true);
    }
  });
  it("rejects invalid or reserved names", () => {
    for (const n of ["ab", "hi there", "-bad", "bad-", "x".repeat(41), "admin", "ABUSE".toLowerCase()]) {
      expect(isAllowedLocal(n)).toBe(false);
    }
  });
});

describe("recipientToAddress", () => {
  it("normalizes an inbound recipient", () => {
    expect(recipientToAddress("Testing@mailbust.com", DOMAIN)).toBe(
      "testing@mailbust.com",
    );
  });

  it("rejects reserved recipients", () => {
    expect(recipientToAddress("admin@mailbust.com", DOMAIN)).toBeNull();
  });
});
