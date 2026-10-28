// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import {
  ACTIONS,
  personURL,
  posthog,
  replayURL,
  unconfigured,
} from "@/server/analytics";

const FROM = new Date("2026-09-17T13:15:00Z");
const TO = new Date("2026-09-17T13:30:00Z");

const row = (counts: number[] = ACTIONS.map(() => 0)): unknown[] => [
  "0192-session",
  "user_a",
  "ada@example.com",
  Date.parse("2026-09-17T12:02:00Z") / 1000,
  Date.parse("2026-09-17T13:20:00Z") / 1000,
  "1.0.0",
  "macos",
  ["/schematic", null, "/lineplot", ""],
  ...counts,
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
    let fetch: ReturnType<typeof vi.fn>;

    beforeEach(() => {
      fetch = vi.fn(async () => Response.json({ results: [row()] }));
      vi.stubGlobal("fetch", fetch);
    });

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    const sent = (): { url: string; init: RequestInit; query: string } => {
      const [url, init] = fetch.mock.calls[0] as [string, RequestInit];
      const body = JSON.parse(init.body as string) as {
        query: { kind: string; query: string };
      };
      expect(body.query.kind).toBe("HogQLQuery");
      return { url, init, query: body.query.query };
    };

    it("should query the Desktop project with the API key", async () => {
      await posthog("phx_key").sessions(FROM, TO);
      const { url, init } = sent();
      expect(url).toBe("https://us.posthog.com/api/projects/621680/query/");
      expect(init.method).toBe("POST");
      expect(init.headers).toMatchObject({ authorization: "Bearer phx_key" });
    });

    it("should select production sessions that end in the range", async () => {
      await posthog("phx_key").sessions(FROM, TO);
      const q = sent().query;
      expect(q).toContain(
        "HAVING max(timestamp) >= toDateTime('2026-09-17 13:15:00', 'UTC') " +
          "AND max(timestamp) < toDateTime('2026-09-17 13:30:00', 'UTC')",
      );
      expect(q).toContain(
        "WHERE timestamp >= toDateTime('2026-09-17 13:15:00', 'UTC') - INTERVAL 1 DAY",
      );
      expect(q).toContain("properties.environment = 'production'");
      for (const a of ACTIONS) expect(q).toContain(`countIf(event = '${a}')`);
    });

    it("should read each row as a session", async () => {
      expect(await posthog("phx_key").sessions(FROM, TO)).toEqual([
        {
          id: "0192-session",
          distinctID: "user_a",
          email: "ada@example.com",
          start: new Date("2026-09-17T12:02:00Z"),
          end: new Date("2026-09-17T13:20:00Z"),
          version: "1.0.0",
          os: "macos",
          screens: ["schematic", "lineplot"],
          actions: {},
        },
      ]);
    });

    it("should keep only the key actions that happened", async () => {
      const counts = ACTIONS.map((a) => (a === "task_started" ? 3 : 0));
      fetch.mockResolvedValue(Response.json({ results: [row(counts)] }));
      const [session] = await posthog("phx_key").sessions(FROM, TO);
      expect(session.actions).toEqual({ task_started: 3 });
    });

    it("should read an empty version or email as none", async () => {
      const r = row();
      r[2] = null;
      r[5] = "";
      fetch.mockResolvedValue(Response.json({ results: [r] }));
      const [session] = await posthog("phx_key").sessions(FROM, TO);
      expect(session.email).toBeNull();
      expect(session.version).toBeNull();
    });

    it("should throw when PostHog refuses the query", async () => {
      fetch.mockResolvedValue(new Response("bad key", { status: 401 }));
      await expect(posthog("phx_key").sessions(FROM, TO)).rejects.toThrow(
        "posthog: query failed with 401: bad key",
      );
    });

    it("should throw on a row of the wrong shape", async () => {
      fetch.mockResolvedValue(Response.json({ results: [["only-an-id"]] }));
      await expect(posthog("phx_key").sessions(FROM, TO)).rejects.toBeInstanceOf(
        z.ZodError,
      );
    });
  });

  describe("unconfigured", () => {
    it("should throw on every read", async () => {
      await expect(unconfigured().sessions(FROM, TO)).rejects.toThrow(
        "POSTHOG_API_KEY is not set",
      );
    });
  });
});
