// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { is, parse } from "@/link/link";

describe("is", () => {
  it("should return true for a simple HTTP URL", () => {
    expect(is("http://example.com")).toBe(true);
  });

  it("should return true for an HTTP URL with a port", () => {
    expect(is("http://example.com:8080")).toBe(true);
  });

  it("should return true for an HTTPS URL with path, query, and fragment", () => {
    expect(is("https://example.com/path?query=param#fragment")).toBe(true);
  });

  it("should return true for an IRI with IPv4", () => {
    expect(is("http://192.168.1.1")).toBe(true);
  });

  it("should return true for an IRI with percent-encoded spaces", () => {
    expect(is("http://example.com/path%20with%20spaces")).toBe(true);
  });

  it("should return true for an IRI with query and fragment", () => {
    expect(is("http://example.com/path?query=123#section")).toBe(true);
  });

  it("should return false for a URL with invalid characters", () => {
    expect(is("http://example.com/invalid|character")).toBe(false);
  });

  it("should return false for a scheme with spaces", () => {
    expect(is("ht tp://example.com")).toBe(false);
  });

  it("should return false for a URL with a missing host", () => {
    expect(is("http://")).toBe(false);
  });

  it("should return false for a URL with an unsupported scheme", () => {
    expect(is("xyz://example.com")).toBe(false);
  });

  it("should return false for an IPv6 with double colons", () => {
    expect(is("http://[2001:db8:::1]")).toBe(false);
  });

  it("should return false for a scheme with a trailing colon", () => {
    expect(is("http:/example.com")).toBe(false);
  });

  it("should return false for a fragment with spaces", () => {
    expect(is("http://example.com#invalid fragment")).toBe(false);
  });
});

describe("parse", () => {
  // Returns each link as its text on screen and its target.
  const links = (raw: string): [string, string][] => {
    const { text, spans } = parse(raw);
    return spans.map(({ start, end, target }) => [text.slice(start, end), target]);
  };

  it("should return a text with no links as it is", () => {
    expect(parse("valve opened at 14 psi")).toEqual({
      text: "valve opened at 14 psi",
      spans: [],
    });
  });

  it("should give the place of a link in the text", () => {
    expect(parse("see https://example.com now")).toEqual({
      text: "see https://example.com now",
      spans: [
        {
          start: 4,
          end: 23,
          kind: "url",
          target: "https://example.com",
          labeled: false,
        },
      ],
    });
  });

  it("should find each kind of link in one text", () => {
    const raw = "10.0.0.1 wrote /var/log/run.log, see https://example.com/run";
    expect(parse(raw).spans.map(({ kind, target }) => [kind, target])).toEqual([
      ["url", "http://10.0.0.1"],
      ["path", "/var/log/run.log"],
      ["url", "https://example.com/run"],
    ]);
  });

  describe("URLs", () => {
    it.each([
      ["an HTTP URL", "http://example.com"],
      ["an HTTPS URL with a path", "https://example.com/a/b?c=1&d=2#e"],
      ["a URL with a port", "http://localhost:9090/health"],
      ["a Synnax link", "synnax://cluster/7e3f/range/a1b2"],
    ])("should link %s", (_, url) => {
      expect(links(`open ${url} now`)).toEqual([[url, url]]);
    });

    it("should open a www address over HTTPS", () => {
      expect(links("go to www.example.com/docs")).toEqual([
        ["www.example.com/docs", "https://www.example.com/docs"],
      ]);
    });

    it.each([
      ["a period", "See https://example.com/a."],
      ["a comma", "https://example.com/a, then stop"],
      ["a question mark", "Is it https://example.com/a?"],
      ["an exclamation mark", "Open https://example.com/a!"],
      ["a colon", "At https://example.com/a: no data"],
      ["parentheses", "(see https://example.com/a)"],
      ["brackets", "[https://example.com/a]"],
      ["braces", "{https://example.com/a}"],
      ["double quotes", '"https://example.com/a"'],
      ["single quotes", "'https://example.com/a'"],
      ["angle brackets", "<https://example.com/a>"],
    ])("should leave %s around a URL out of the link", (_, raw) => {
      expect(links(raw)).toEqual([["https://example.com/a", "https://example.com/a"]]);
    });

    it("should keep parentheses that close inside a URL", () => {
      const url = "https://en.wikipedia.org/wiki/Rocket_(disambiguation)";
      expect(links(`(see ${url}).`)).toEqual([[url, url]]);
    });

    it.each([
      ["a URL of another scheme", "ftp://example.com/file"],
      ["a domain with no scheme", "example.com"],
    ])("should not link %s", (_, text) => {
      expect(parse(`get ${text} now`).spans).toEqual([]);
    });
  });

  describe("IP addresses", () => {
    it.each([
      ["an address", "10.0.0.1"],
      ["an address with a port", "10.0.0.1:9090"],
      ["an address with a path", "192.168.1.20/metrics"],
      ["an address with a port and a path", "10.0.0.1:9090/api/v1/health"],
    ])("should open %s over HTTP", (_, address) => {
      expect(links(`rack at ${address} is up`)).toEqual([
        [address, `http://${address}`],
      ]);
    });

    it("should link an address that starts the text", () => {
      expect(parse("10.0.0.1 is up").spans).toMatchObject([{ start: 0, end: 8 }]);
    });

    it("should leave the period of a sentence out of the link", () => {
      expect(links("Lost 10.0.0.1.")).toEqual([["10.0.0.1", "http://10.0.0.1"]]);
    });

    it.each([
      ["a version", "v1.2.3.4"],
      ["a number with five parts", "1.2.3.4.5"],
      ["a number with three parts", "1.2.3"],
      ["a part above 255", "10.0.0.256"],
      ["a subnet", "10.0.0.0/24"],
      ["an address of another scheme", "tcp://10.0.0.1:502"],
    ])("should not link %s", (_, text) => {
      expect(parse(`found ${text} here`).spans).toEqual([]);
    });
  });

  describe("file paths", () => {
    it.each([
      ["a POSIX path", "/var/log/synnax/run-1.log"],
      ["a path in the home directory", "~/logs/run-1.log"],
      ["a Windows path", "C:\\Users\\synnax\\run-1.log"],
      ["a Windows path with forward slashes", "C:/Users/synnax/run-1.log"],
    ])("should link %s", (_, path) => {
      expect(parse(`wrote ${path} to disk`).spans).toEqual([
        { start: 6, end: 6 + path.length, kind: "path", target: path, labeled: false },
      ]);
    });

    it("should link a path that starts the text", () => {
      expect(parse("/var/log/run.log is full").spans).toMatchObject([
        { start: 0, end: 16, kind: "path" },
      ]);
    });

    it.each([
      ["an equals sign", "log_path=/var/log/run.log"],
      ["an opening parenthesis", "(/var/log/run.log)"],
      ["an opening bracket", "[/var/log/run.log]"],
      ["a quote", 'cannot read "/var/log/run.log"'],
    ])("should link a path after %s", (_, raw) => {
      expect(links(raw)).toEqual([["/var/log/run.log", "/var/log/run.log"]]);
    });

    it("should leave the period of a sentence out of the link", () => {
      expect(links("See /var/log/run.log.")).toEqual([
        ["/var/log/run.log", "/var/log/run.log"],
      ]);
    });

    it("should end a path at a space", () => {
      expect(links("open /var/log/my run.log")).toEqual([
        ["/var/log/my", "/var/log/my"],
      ]);
    });

    it.each([
      ["a slash between words", "and/or"],
      ["a fraction", "1/2"],
      ["a unit", "50 m/s"],
      ["a lone slash", "a / b"],
      ["a relative path", "logs/run.log"],
    ])("should not link %s", (_, text) => {
      expect(parse(`with ${text} here`).spans).toEqual([]);
    });

    // To open a UNC path sends the login of the user to its host.
    it.each([
      ["with backslashes", "\\\\server\\share\\run.log"],
      ["with forward slashes", "//server/share/run.log"],
    ])("should not link a UNC path %s", (_, path) => {
      expect(parse(`open ${path} now`).spans).toEqual([]);
    });
  });

  describe("Markdown links", () => {
    it("should show a Markdown link as its label", () => {
      expect(parse("see [the docs](https://docs.synnaxlabs.com) for more")).toEqual({
        text: "see the docs for more",
        spans: [
          {
            start: 4,
            end: 12,
            kind: "url",
            target: "https://docs.synnaxlabs.com",
            labeled: true,
          },
        ],
      });
    });

    it.each([
      ["a path", "[the log](/var/log/run.log)", "the log", "path", "/var/log/run.log"],
      [
        "an IP address",
        "[rack 1](10.0.0.1:9090)",
        "rack 1",
        "url",
        "http://10.0.0.1:9090",
      ],
      [
        "a www address",
        "[site](www.example.com)",
        "site",
        "url",
        "https://www.example.com",
      ],
      [
        "a Synnax link",
        "[range](synnax://range/a1b2)",
        "range",
        "url",
        "synnax://range/a1b2",
      ],
    ])("should label a Markdown link to %s", (_, raw, text, kind, target) => {
      expect(parse(raw)).toEqual({
        text,
        spans: [{ start: 0, end: text.length, kind, target, labeled: true }],
      });
    });

    it("should place each link in the text as it shows", () => {
      const { text, spans } = parse(
        "[a](https://a.example.com) then [b](https://b.example.com) or https://c.example.com",
      );
      expect(text).toBe("a then b or https://c.example.com");
      expect(
        spans.map(({ start, end, labeled }) => [text.slice(start, end), labeled]),
      ).toEqual([
        ["a", true],
        ["b", true],
        ["https://c.example.com", false],
      ]);
    });

    it("should keep parentheses that close inside the target", () => {
      const target = "https://en.wikipedia.org/wiki/Rocket_(disambiguation)";
      expect(parse(`[rocket](${target})`)).toEqual({
        text: "rocket",
        spans: [{ start: 0, end: 6, kind: "url", target, labeled: true }],
      });
    });

    it("should keep bracket text that is not the label", () => {
      expect(parse("[warn] [docs](https://example.com)").text).toBe("[warn] docs");
    });

    it.each([
      ["a link in parentheses", "(https://example.com)"],
      ["a link after bracket text", "[warn] https://example.com"],
      ["a space between the label and the link", "[docs] (https://example.com)"],
      ["an empty label", "[](https://example.com)"],
    ])("should show %s as it arrives", (_, raw) => {
      expect(parse(raw)).toMatchObject({
        text: raw,
        spans: [{ target: "https://example.com", labeled: false }],
      });
    });

    it("should not hide a target behind a label that is a link", () => {
      const raw = "[https://bank.example.com](https://evil.example.com)";
      const { text, spans } = parse(raw);
      expect(text).toBe(raw);
      expect(spans.some(({ labeled }) => labeled)).toBe(false);
    });
  });
});
