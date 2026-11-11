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
    const SUBSCRIBED = { unsubscribed: false, everyone: true, account: true };

    it("should add every account to the list in lowercase", async () => {
      const audience = memory();
      expect(await sync(directory({ people: { a: ADA, g: GRACE } }), audience)).toEqual(
        { added: 2, removed: 0 },
      );
      expect(Object.fromEntries(audience.contacts)).toEqual({
        "ada@example.com": SUBSCRIBED,
        "grace@example.com": SUBSCRIBED,
      });
    });

    it("should leave an address added by hand as it is", async () => {
      const audience = memory(["ada@example.com"]);
      expect(await sync(directory({ people: { a: ADA } }), audience)).toEqual({
        added: 0,
        removed: 0,
      });
      expect(audience.contacts.get("ada@example.com")?.account).toBe(false);
    });

    it("should keep a hand-added address after its account is deleted", async () => {
      const audience = memory(["ada@example.com"]);
      await sync(directory({ people: { a: ADA } }), audience);
      await sync(directory(), audience);
      expect([...audience.contacts.keys()]).toEqual(["ada@example.com"]);
    });

    it("should delete the address of a deleted account", async () => {
      const audience = memory();
      await sync(directory({ people: { a: ADA, g: GRACE } }), audience);
      expect(await sync(directory({ people: { g: GRACE } }), audience)).toEqual({
        added: 0,
        removed: 1,
      });
      expect([...audience.contacts.keys()]).toEqual(["grace@example.com"]);
    });

    it("should keep an unsubscribe when an account leaves and comes back", async () => {
      const audience = memory();
      await sync(directory({ people: { a: ADA } }), audience);
      audience.contacts.set("ada@example.com", { ...SUBSCRIBED, unsubscribed: true });
      await sync(directory(), audience);
      expect(audience.contacts.get("ada@example.com")).toEqual({
        unsubscribed: true,
        everyone: false,
        account: false,
      });
      await sync(directory({ people: { a: ADA } }), audience);
      expect(audience.contacts.get("ada@example.com")).toEqual({
        ...SUBSCRIBED,
        unsubscribed: true,
      });
    });

    it("should move an account to its new address", async () => {
      const audience = memory();
      await sync(directory({ people: { a: ADA } }), audience);
      await sync(
        directory({ people: { a: { ...ADA, email: "ada@acme.co" } } }),
        audience,
      );
      expect([...audience.contacts.keys()]).toEqual(["ada@acme.co"]);
    });

    it("should skip an account with no address", async () => {
      const audience = memory();
      await sync(directory({ people: { a: { ...ADA, email: "" } } }), audience);
      expect(audience.contacts.size).toBe(0);
    });
  });

  describe("resend", () => {
    const SEGMENTS = {
      object: "list",
      has_more: false,
      data: [
        { id: "seg_all", name: "Release notes" },
        { id: "seg_accounts", name: "Portal accounts" },
      ],
    };
    const ADDRESS = "/contacts/ada@example.com";
    const ok = (): Response => Response.json({ object: "contact", id: "id_ada" });
    const contact = (unsubscribed: boolean) => (): Response =>
      Response.json({ object: "contact", id: "id_ada", unsubscribed, properties: {} });
    const refused =
      (name: string, status: number, headers = {}) =>
      (): Response =>
        Response.json(
          { name, message: `${name} message`, statusCode: status },
          {
            status,
            headers,
          },
        );

    // routes answers a request by its method and path, as `GET /segments`.
    let routes: Record<string, (url: URL) => Response>;
    // The Resend SDK always calls fetch with a string URL.
    let fetch: Mock<(url: string, init?: RequestInit) => Promise<Response>>;
    const requests = (): string[] =>
      fetch.mock.calls.map(
        ([url, init]) => `${init?.method ?? "GET"} ${new URL(url).pathname}`,
      );

    beforeEach(() => {
      routes = { "GET /segments": () => Response.json(SEGMENTS) };
      fetch = vi.fn(async (url: string, init?: RequestInit) => {
        const parsed = new URL(url);
        const route = `${init?.method ?? "GET"} ${parsed.pathname}`;
        const answer = routes[route];
        if (answer == null) throw new Error(`no route for ${route}`);
        return answer(parsed);
      });
      vi.stubGlobal("fetch", fetch);
      vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
      vi.unstubAllGlobals();
      vi.restoreAllMocks();
    });

    it("should read every page of a segment in lowercase", async () => {
      routes["GET /segments/seg_all/contacts"] = (url) =>
        Response.json({
          object: "list",
          has_more: url.searchParams.get("after") == null,
          data:
            url.searchParams.get("after") == null
              ? [{ id: "id_ada", email: "Ada@Example.com" }]
              : [{ id: "id_grace", email: "grace@example.com" }],
        });
      expect(await resend("re_key").everyone()).toEqual(
        new Set(["ada@example.com", "grace@example.com"]),
      );
      expect(new URL(fetch.mock.calls[2][0]).searchParams.get("after")).toBe("id_ada");
    });

    it("should create a new address in both segments", async () => {
      routes[`GET ${ADDRESS}`] = refused("not_found", 404);
      routes["POST /contacts"] = ok;
      await resend("re_key").add("ada@example.com");
      const [, init] = fetch.mock.calls[2];
      expect(JSON.parse(init?.body as string)).toMatchObject({
        email: "ada@example.com",
        segments: [{ id: "seg_all" }, { id: "seg_accounts" }],
      });
    });

    it("should put a known address back in both segments", async () => {
      routes[`GET ${ADDRESS}`] = contact(true);
      routes[`POST ${ADDRESS}/segments/seg_all`] = ok;
      routes[`POST ${ADDRESS}/segments/seg_accounts`] = ok;
      await resend("re_key").add("ada@example.com");
      expect(requests()).toEqual([
        "GET /segments",
        `GET ${ADDRESS}`,
        `POST ${ADDRESS}/segments/seg_all`,
        `POST ${ADDRESS}/segments/seg_accounts`,
      ]);
    });

    it("should delete an address that did not unsubscribe", async () => {
      routes[`GET ${ADDRESS}`] = contact(false);
      routes[`DELETE ${ADDRESS}`] = ok;
      await resend("re_key").remove("ada@example.com");
      expect(requests()).toEqual([`GET ${ADDRESS}`, `DELETE ${ADDRESS}`]);
    });

    it("should keep an unsubscribed address outside the segments", async () => {
      routes[`GET ${ADDRESS}`] = contact(true);
      routes[`DELETE ${ADDRESS}/segments/seg_all`] = ok;
      routes[`DELETE ${ADDRESS}/segments/seg_accounts`] = ok;
      await resend("re_key").remove("ada@example.com");
      expect(requests()).toEqual([
        `GET ${ADDRESS}`,
        "GET /segments",
        `DELETE ${ADDRESS}/segments/seg_all`,
        `DELETE ${ADDRESS}/segments/seg_accounts`,
      ]);
    });

    it("should retry a request after the time Resend asks for", async () => {
      routes[`GET ${ADDRESS}`] = contact(false);
      routes[`DELETE ${ADDRESS}`] = ok;
      fetch.mockImplementationOnce(async () =>
        refused("rate_limit_exceeded", 429, { "retry-after": "0.01" })(),
      );
      await resend("re_key").remove("ada@example.com");
      expect(requests()).toEqual([
        `GET ${ADDRESS}`,
        `GET ${ADDRESS}`,
        `DELETE ${ADDRESS}`,
      ]);
    });

    it("should throw when Resend refuses a request", async () => {
      routes[`GET ${ADDRESS}`] = refused("restricted_api_key", 401);
      await expect(resend("re_key").remove("ada@example.com")).rejects.toThrow(
        "audience: restricted_api_key message",
      );
    });

    it("should throw when a segment is missing", async () => {
      routes["GET /segments"] = () =>
        Response.json({ ...SEGMENTS, data: [SEGMENTS.data[0]] });
      await expect(resend("re_key").accounts()).rejects.toThrow(
        "audience: no Resend segment Portal accounts",
      );
    });
  });
});
