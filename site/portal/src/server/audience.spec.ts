// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { memory, resend, sync } from "@/server/audience";
import { memory as directory } from "@/server/directory";

const ADA = { name: "Ada Lovelace", email: "Ada@Example.com" };
const GRACE = { name: "Grace Hopper", email: "grace@example.com" };

describe("audience", () => {
  describe("sync", () => {
    it("should add every account to the list in lowercase", async () => {
      const audience = memory();
      expect(await sync(directory({ people: { a: ADA, g: GRACE } }), audience)).toEqual(
        { added: 2, removed: 0 },
      );
      expect([...audience.list].sort()).toEqual([
        "ada@example.com",
        "grace@example.com",
      ]);
      expect(audience.synced).toEqual(audience.list);
    });

    it("should leave an address added by hand as it is", async () => {
      const audience = memory(["ada@example.com"]);
      expect(await sync(directory({ people: { a: ADA } }), audience)).toEqual({
        added: 0,
        removed: 0,
      });
      expect(audience.synced.size).toBe(0);
    });

    it("should keep a hand-added address after its account is deleted", async () => {
      const audience = memory(["ada@example.com"]);
      await sync(directory({ people: { a: ADA } }), audience);
      await sync(directory(), audience);
      expect([...audience.list]).toEqual(["ada@example.com"]);
    });

    it("should remove the address of a deleted account", async () => {
      const audience = memory();
      await sync(directory({ people: { a: ADA, g: GRACE } }), audience);
      expect(await sync(directory({ people: { g: GRACE } }), audience)).toEqual({
        added: 0,
        removed: 1,
      });
      expect([...audience.list]).toEqual(["grace@example.com"]);
    });

    it("should move an account to its new address", async () => {
      const audience = memory();
      await sync(directory({ people: { a: ADA } }), audience);
      await sync(
        directory({ people: { a: { ...ADA, email: "ada@acme.co" } } }),
        audience,
      );
      expect([...audience.list]).toEqual(["ada@acme.co"]);
    });

    it("should skip an account with no address", async () => {
      const audience = memory();
      await sync(directory({ people: { a: { ...ADA, email: "" } } }), audience);
      expect(audience.list.size).toBe(0);
    });
  });

  describe("resend", () => {
    const API = "https://api.resend.com";
    const SEGMENTS = {
      object: "list",
      has_more: false,
      data: [
        { id: "seg_all", name: "Release notes" },
        { id: "seg_accounts", name: "Portal accounts" },
      ],
    };
    // The Resend SDK always calls fetch with a string URL.
    let fetch: Mock<(url: string, init?: RequestInit) => Promise<Response>>;

    const contacts = (emails: string[], more: boolean): Response =>
      Response.json({
        object: "list",
        has_more: more,
        data: emails.map((email) => ({ id: `id_${email}`, email })),
      });

    beforeEach(() => {
      fetch = vi.fn(async (url: string) =>
        url === `${API}/segments`
          ? Response.json(SEGMENTS)
          : Response.json({ object: "contact", id: "id_new" }),
      );
      vi.stubGlobal("fetch", fetch);
      vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
      vi.unstubAllGlobals();
      vi.restoreAllMocks();
    });

    it("should read every page of a segment in lowercase", async () => {
      fetch.mockImplementation(async (url) => {
        const { pathname, searchParams } = new URL(url);
        if (pathname === "/segments") return Response.json(SEGMENTS);
        expect(pathname).toBe("/segments/seg_all/contacts");
        return searchParams.get("after") == null
          ? contacts(["Ada@Example.com"], true)
          : contacts(["grace@example.com"], false);
      });
      expect(await resend("re_key").everyone()).toEqual(
        new Set(["ada@example.com", "grace@example.com"]),
      );
      expect(new URL(fetch.mock.calls[2][0]).searchParams.get("after")).toBe(
        "id_Ada@Example.com",
      );
    });

    it("should add an address to both segments", async () => {
      await resend("re_key").add("ada@example.com");
      const [url, init] = fetch.mock.calls[1];
      expect(url).toBe(`${API}/contacts`);
      expect(JSON.parse(init?.body as string)).toMatchObject({
        email: "ada@example.com",
        segments: [{ id: "seg_all" }, { id: "seg_accounts" }],
      });
    });

    it("should remove an address", async () => {
      await resend("re_key").remove("ada@example.com");
      const [url, init] = fetch.mock.calls[0];
      expect(url).toBe(`${API}/contacts/ada@example.com`);
      expect(init?.method).toBe("DELETE");
    });

    it("should retry a request that Resend refused for its rate", async () => {
      fetch.mockResolvedValueOnce(
        Response.json(
          {
            name: "rate_limit_exceeded",
            message: "Too many requests",
            statusCode: 429,
          },
          { status: 429, headers: { "retry-after": "0" } },
        ),
      );
      await resend("re_key").remove("ada@example.com");
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should throw when Resend refuses a request", async () => {
      fetch.mockResolvedValue(
        Response.json(
          { name: "restricted_api_key", message: "Key cannot read", statusCode: 401 },
          { status: 401 },
        ),
      );
      await expect(resend("re_key").remove("ada@example.com")).rejects.toThrow(
        "audience: Key cannot read",
      );
    });

    it("should throw when a segment is missing", async () => {
      fetch.mockResolvedValue(Response.json({ ...SEGMENTS, data: [SEGMENTS.data[0]] }));
      await expect(resend("re_key").accounts()).rejects.toThrow(
        "audience: no Resend segment Portal accounts",
      );
    });
  });
});
