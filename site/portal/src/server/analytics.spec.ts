// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import { z } from "zod";

import {
  ACTIONS,
  type Capture,
  personURL,
  posthog,
  replayURL,
  unconfigured,
} from "@/server/analytics";

const FROM = new Date("2026-09-17T13:15:00Z");
const TO = new Date("2026-09-17T13:30:00Z");

interface Row {
  id?: string;
  distinctID?: string;
  email?: string | null;
  start?: string;
  end?: string;
  version?: string | null;
  paths?: (string | null)[];
  actions?: Partial<Record<(typeof ACTIONS)[number], number>>;
}

const row = ({
  id = "0192-session",
  distinctID = "user_a",
  email = "ada@example.com",
  start = "2026-09-17T12:02:00Z",
  end = "2026-09-17T13:20:00Z",
  version = "1.0.0",
  paths = ["/schematic", null, "/lineplot", ""],
  actions = {},
}: Row = {}): unknown[] => [
  id,
  distinctID,
  email,
  Date.parse(start) / 1000,
  Date.parse(end) / 1000,
  version,
  "macos",
  paths,
  ...ACTIONS.map((a) => actions[a] ?? 0),
];

describe("analytics", () => {
  describe("URLs", () => {
    it("should open a person and a replay in the Desktop project", () => {
      expect(personURL("user_a")).toBe(
        "https://us.posthog.com/project/621680/person/user_a",
      );
      expect(replayURL("0192 session")).toBe(
        "https://us.posthog.com/project/621680/replay/0192%20session",
      );
    });
  });

  describe("posthog", () => {
    let fetch: Mock<typeof globalThis.fetch>;

    beforeEach(() => {
      fetch = vi.fn<typeof globalThis.fetch>(async () =>
        Response.json({ results: [row()] }),
      );
      vi.stubGlobal("fetch", fetch);
    });

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    const respond = (...rows: Row[]): void => {
      fetch.mockResolvedValue(Response.json({ results: rows.map(row) }));
    };

    const visits = async () =>
      await posthog({ apiKey: "phx_key", environment: "production" }).visits(FROM, TO);

    const sent = (): { url: string; init: RequestInit; query: string } => {
      const [url, init] = fetch.mock.calls[0] as [string, RequestInit];
      const body = JSON.parse(init.body as string) as {
        query: { kind: string; query: string };
      };
      expect(body.query.kind).toBe("HogQLQuery");
      return { url, init, query: body.query.query };
    };

    it("should query the Desktop project with the API key", async () => {
      await visits();
      const { url, init } = sent();
      expect(url).toBe("https://us.posthog.com/api/projects/621680/query/");
      expect(init.method).toBe("POST");
      expect(init.headers).toMatchObject({ authorization: "Bearer phx_key" });
    });

    it("should read the production sessions of the day before the range", async () => {
      await visits();
      const q = sent().query;
      expect(q).toContain(
        "WHERE timestamp >= toDateTime('2026-09-16 13:15:00', 'UTC')",
      );
      expect(q).toContain("properties.environment = 'production'");
      expect(q).toContain("GROUP BY $session_id");
      for (const a of ACTIONS) expect(q).toContain(`countIf(event = '${a}')`);
    });

    it("should read a session that ended in the range as a visit", async () => {
      expect(await visits()).toEqual([
        {
          distinctID: "user_a",
          email: "ada@example.com",
          start: new Date("2026-09-17T12:02:00Z"),
          end: new Date("2026-09-17T13:20:00Z"),
          version: "1.0.0",
          os: "macos",
          screens: ["schematic", "lineplot"],
          actions: {},
          replay: "0192-session",
        },
      ]);
    });

    it("should keep only the key actions that happened", async () => {
      respond({ actions: { task_started: 3 } });
      const [visit] = await visits();
      expect(visit.actions).toEqual({ task_started: 3 });
    });

    it("should read an empty version or email as none", async () => {
      respond({ email: null, version: "" });
      const [visit] = await visits();
      expect(visit.email).toBeNull();
      expect(visit.version).toBeNull();
    });

    it("should join the sessions of each window of one person", async () => {
      respond(
        { id: "main", actions: { plot_created: 1 }, paths: ["/lineplot"] },
        {
          id: "task-window",
          start: "2026-09-17T13:19:00Z",
          version: null,
          paths: ["/task"],
          actions: { plot_created: 2, task_started: 1 },
        },
        { id: "reset", start: "2026-09-17T13:20:00Z", email: null, paths: [] },
      );
      expect(await visits()).toEqual([
        {
          distinctID: "user_a",
          email: "ada@example.com",
          start: new Date("2026-09-17T12:02:00Z"),
          end: new Date("2026-09-17T13:20:00Z"),
          version: "1.0.0",
          os: "macos",
          screens: ["lineplot", "task"],
          actions: { plot_created: 3, task_started: 1 },
          replay: "main",
        },
      ]);
    });

    it("should join a session that starts before the idle wait ends", async () => {
      respond(
        { id: "morning", start: "2026-09-17T11:00:00Z", end: "2026-09-17T12:00:00Z" },
        { id: "noon", start: "2026-09-17T12:29:00Z" },
      );
      const [visit] = await visits();
      expect(visit.start).toEqual(new Date("2026-09-17T11:00:00Z"));
      expect(visit.replay).toBe("morning");
    });

    it("should keep apart sessions more than the idle wait apart", async () => {
      respond(
        { id: "morning", start: "2026-09-17T11:00:00Z", end: "2026-09-17T12:00:00Z" },
        { id: "noon", start: "2026-09-17T12:30:00Z" },
      );
      expect((await visits()).map((v) => v.replay)).toEqual(["noon"]);
    });

    it("should keep the visits of different people apart, oldest first", async () => {
      respond(
        { id: "bob", distinctID: "user_b", start: "2026-09-17T12:10:00Z" },
        { id: "ada" },
      );
      expect((await visits()).map((v) => v.replay)).toEqual(["ada", "bob"]);
    });

    it("should leave out a visit that goes on past the range", async () => {
      respond(
        { id: "main" },
        { id: "late", start: "2026-09-17T13:25:00Z", end: "2026-09-17T13:40:00Z" },
      );
      expect(await visits()).toEqual([]);
    });

    /** sinceOf reads the start of the events that a query reads. */
    const sinceOf = (init?: RequestInit): string => {
      const body = JSON.parse(init?.body as string) as { query: { query: string } };
      const since = /timestamp >= toDateTime\('([^']+)'/.exec(body.query.query);
      expect(since).not.toBeNull();
      return since?.[1] ?? "";
    };

    const readFrom = (): string[] => fetch.mock.calls.map(([, init]) => sinceOf(init));

    it("should read back further for a visit that began before the day it read", async () => {
      fetch.mockImplementation(async (_, init) => {
        const start =
          sinceOf(init) === "2026-09-16 13:15:00"
            ? "2026-09-16T13:16:00Z"
            : "2026-09-16T10:00:00Z";
        return Response.json({ results: [row({ start })] });
      });
      const [visit] = await visits();
      expect(visit.start).toEqual(new Date("2026-09-16T10:00:00Z"));
      expect(readFrom()).toEqual(["2026-09-16 13:15:00", "2026-09-15 13:15:00"]);
    });

    it("should read back at most 8 days", async () => {
      fetch.mockImplementation(async (_, init) => {
        const since = Date.parse(`${sinceOf(init).replace(" ", "T")}Z`);
        const start = new Date(since + 60_000).toISOString();
        return Response.json({ results: [row({ start })] });
      });
      const [visit] = await visits();
      expect(readFrom()).toEqual([
        "2026-09-16 13:15:00",
        "2026-09-15 13:15:00",
        "2026-09-13 13:15:00",
        "2026-09-09 13:15:00",
      ]);
      expect(visit.start).toEqual(new Date("2026-09-09T13:16:00Z"));
    });

    it("should throw when PostHog refuses the query", async () => {
      fetch.mockResolvedValue(new Response("bad key", { status: 401 }));
      await expect(visits()).rejects.toThrow("posthog: query failed with 401: bad key");
    });

    it("should throw on a row of the wrong shape", async () => {
      fetch.mockResolvedValue(Response.json({ results: [["only-an-id"]] }));
      await expect(visits()).rejects.toBeInstanceOf(z.ZodError);
    });
  });

  describe("capture", () => {
    let fetch: Mock<typeof globalThis.fetch>;

    beforeEach(() => {
      fetch = vi.fn<typeof globalThis.fetch>(async () => Response.json({ status: 1 }));
      vi.stubGlobal("fetch", fetch);
    });

    afterEach(() => {
      vi.unstubAllGlobals();
      vi.restoreAllMocks();
    });

    const EVENT: Capture = {
      userID: "user_a",
      event: "referral_converted",
      at: FROM,
      properties: { referral: "abc" },
      once: { referral: "abc" },
    };

    const capture = async (environment: "production" | "development" = "production") =>
      await posthog({ apiKey: "phx_key", environment }).capture(EVENT);

    it("should send the event with the project key, as the user", async () => {
      await capture();
      const [url, init] = fetch.mock.calls[0] as [string, RequestInit];
      expect(url).toBe("https://us.i.posthog.com/i/v0/e/");
      expect(JSON.parse(init.body as string)).toEqual({
        api_key: "phc_CaKeC8DVDWCYaiYSnUpEaWWXLrWptuXwgF7tTY3U6Q7t",
        event: "referral_converted",
        distinct_id: "user_a",
        timestamp: "2026-09-17T13:15:00.000Z",
        properties: {
          referral: "abc",
          environment: "production",
          $set_once: { referral: "abc" },
        },
      });
    });

    it("should tag the event with the environment", async () => {
      await capture("development");
      const [, init] = fetch.mock.calls[0] as [string, RequestInit];
      expect(JSON.parse(init.body as string)).toMatchObject({
        properties: { environment: "development" },
      });
    });

    it("should log a refused event instead of throwing", async () => {
      const logged = vi.spyOn(console, "error").mockImplementation(() => {});
      fetch.mockResolvedValue(new Response("bad token", { status: 401 }));
      await capture();
      expect(logged).toHaveBeenCalledWith("posthog capture failed", {
        event: "referral_converted",
        status: 401,
        body: "bad token",
      });
    });

    it("should log a failed request instead of throwing", async () => {
      const logged = vi.spyOn(console, "error").mockImplementation(() => {});
      const err = new Error("offline");
      fetch.mockRejectedValue(err);
      await capture();
      expect(logged).toHaveBeenCalledWith(
        "posthog capture failed",
        { event: "referral_converted" },
        err,
      );
    });
  });

  describe("unconfigured", () => {
    it("should throw on every read", async () => {
      await expect(unconfigured().visits(FROM, TO)).rejects.toThrow(
        "POSTHOG_API_KEY is not set",
      );
    });

    it("should log a capture instead of sending it", async () => {
      const fetch = vi.fn<typeof globalThis.fetch>();
      vi.stubGlobal("fetch", fetch);
      const logged = vi.spyOn(console, "info").mockImplementation(() => {});
      await unconfigured().capture({
        userID: "user_a",
        event: "referral_converted",
        at: FROM,
        properties: {},
        once: {},
      });
      expect(fetch).not.toHaveBeenCalled();
      expect(logged).toHaveBeenCalledWith("analytics event not sent", {
        event: "referral_converted",
      });
      vi.unstubAllGlobals();
      vi.restoreAllMocks();
    });
  });
});
