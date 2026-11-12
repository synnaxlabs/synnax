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
    const SUBSCRIBED = { unsubscribed: false, segments: ["accounts", "everyone"] };

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
      expect(audience.contacts.get("ada@example.com")?.segments).toEqual(["everyone"]);
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
        segments: [],
      });
      await sync(directory({ people: { a: ADA } }), audience);
      expect(audience.contacts.get("ada@example.com")).toEqual({
        ...SUBSCRIBED,
        unsubscribed: true,
      });
    });

    it("should keep a contact that another segment holds", async () => {
      const audience = memory();
      audience.contacts.set("ada@example.com", {
        unsubscribed: false,
        segments: ["events"],
      });
      await sync(directory({ people: { a: ADA } }), audience);
      await sync(directory(), audience);
      expect(audience.contacts.get("ada@example.com")).toEqual({
        unsubscribed: false,
        segments: ["events"],
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
        segments: [{ id: "seg_accounts" }, { id: "seg_all" }],
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
        `POST ${ADDRESS}/segments/seg_accounts`,
        `POST ${ADDRESS}/segments/seg_all`,
      ]);
    });

    const held =
      (...ids: string[]) =>
      (): Response =>
        Response.json({
          object: "list",
          has_more: false,
          data: ids.map((id) => ({ id, name: id })),
        });

    it("should delete an address that only the list held", async () => {
      routes[`GET ${ADDRESS}`] = contact(false);
      routes[`GET ${ADDRESS}/segments`] = held("seg_all", "seg_accounts");
      routes[`DELETE ${ADDRESS}`] = ok;
      await resend("re_key").remove("ada@example.com");
      expect(requests().at(-1)).toBe(`DELETE ${ADDRESS}`);
    });

    it("should keep an unsubscribed address outside the segments", async () => {
      routes[`GET ${ADDRESS}`] = contact(true);
      routes[`GET ${ADDRESS}/segments`] = held("seg_all", "seg_accounts");
      routes[`DELETE ${ADDRESS}/segments/seg_all`] = ok;
      routes[`DELETE ${ADDRESS}/segments/seg_accounts`] = ok;
      await resend("re_key").remove("ada@example.com");
      expect(requests().slice(-2)).toEqual([
        `DELETE ${ADDRESS}/segments/seg_all`,
        `DELETE ${ADDRESS}/segments/seg_accounts`,
      ]);
    });

    it("should keep an address that another segment holds", async () => {
      routes[`GET ${ADDRESS}`] = contact(false);
      routes[`GET ${ADDRESS}/segments`] = held("seg_all", "seg_accounts", "seg_events");
      routes[`DELETE ${ADDRESS}/segments/seg_all`] = ok;
      routes[`DELETE ${ADDRESS}/segments/seg_accounts`] = ok;
      await resend("re_key").remove("ada@example.com");
      expect(requests()).not.toContain(`DELETE ${ADDRESS}`);
    });

    const BROADCAST = {
      name: "v0.61.0",
      from: "Ada <ada@synnaxlabs.com>",
      subject: "Synnax v0.61",
      html: "<p>Hi</p>",
      text: "Hi",
    };
    const broadcasts =
      (...sent: Array<{ name: string; status: string }>) =>
      (url: URL): Response =>
        Response.json({
          object: "list",
          has_more: url.searchParams.get("after") == null,
          data:
            url.searchParams.get("after") == null
              ? [{ id: "b_old", name: "v0.60.0", status: "sent" }]
              : sent.map((b, i) => ({ id: `b_${i}`, ...b })),
        });

    it("should send a broadcast to everyone on the list", async () => {
      routes["GET /broadcasts"] = broadcasts({ name: "v0.61.0", status: "draft" });
      routes["POST /broadcasts"] = () => Response.json({ id: "b_new" });
      await resend("re_key").send(BROADCAST);
      const [, init] = fetch.mock.calls.at(-1) ?? [];
      expect(JSON.parse(init?.body as string)).toMatchObject({
        ...BROADCAST,
        segment_id: "seg_all",
        send: true,
      });
    });

    it("should refuse a broadcast whose name was sent before", async () => {
      routes["GET /broadcasts"] = broadcasts({ name: "v0.61.0", status: "sent" });
      await expect(resend("re_key").send(BROADCAST)).rejects.toThrow(
        "audience: v0.61.0 was sent before",
      );
      expect(requests()).not.toContain("POST /broadcasts");
    });

    it("should retry a request after the time Resend asks for", async () => {
      routes["GET /segments/seg_accounts/contacts"] = held();
      fetch.mockImplementationOnce(async () =>
        refused("rate_limit_exceeded", 429, { "retry-after": "0.01" })(),
      );
      expect(await resend("re_key").accounts()).toEqual(new Set());
      expect(requests()).toEqual([
        "GET /segments",
        "GET /segments",
        "GET /segments/seg_accounts/contacts",
      ]);
    });

    it("should throw when Resend refuses a request", async () => {
      routes[`GET ${ADDRESS}`] = refused("restricted_api_key", 401);
      await expect(resend("re_key").add("ada@example.com")).rejects.toThrow(
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
