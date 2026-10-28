// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { type Visit } from "@/server/analytics";
import { type Organization } from "@/server/db/schema";
import { LICENSE, NOW } from "@/server/license/testutil";
import {
  active,
  announce,
  digest,
  firstLogin,
  joined,
  memory,
  type Note,
  recap,
  refused,
  webhook,
} from "@/server/slack";

const ORIGIN = "https://portal.synnaxlabs.com";
const URL = "https://hooks.slack.com/services/T0/B0/secret";

const ADA = { userID: "user_a", name: "Ada Lovelace", email: "ada@example.com" };

const NOTE: Note = {
  title: "New account",
  facts: [{ label: "Name", value: "Ada <Lovelace> & Co" }],
  links: [{ label: "Open the user", href: "/admin/users/user_a" }],
};

describe("slack", () => {
  describe("webhook", () => {
    let fetch: ReturnType<typeof vi.fn>;
    let error: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
      fetch = vi.fn(async () => new Response("ok"));
      vi.stubGlobal("fetch", fetch);
      error = vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
      vi.unstubAllGlobals();
      vi.restoreAllMocks();
    });

    const posted = async (note: Note): Promise<string> => {
      await webhook({ url: URL, origin: ORIGIN }).post(note);
      expect(fetch).toHaveBeenCalledOnce();
      const [url, init] = fetch.mock.calls[0] as [string, RequestInit];
      expect(url).toBe(URL);
      expect(init.method).toBe("POST");
      return (JSON.parse(init.body as string) as { text: string }).text;
    };

    it("should post the title, the facts, and the link as escaped markup", async () => {
      expect(await posted(NOTE)).toBe(
        [
          "*New account*",
          "Name: Ada &lt;Lovelace&gt; &amp; Co",
          "<https://portal.synnaxlabs.com/admin/users/user_a|Open the user>",
        ].join("\n"),
      );
      expect(error).not.toHaveBeenCalled();
    });

    it("should keep a full URL and join several links", async () => {
      const links = [
        { label: "Open the user", href: "/admin/users/user_a" },
        { label: "Watch the replay", href: "https://us.posthog.com/replay/s" },
      ];
      expect((await posted({ ...NOTE, links })).split("\n").at(-1)).toBe(
        "<https://portal.synnaxlabs.com/admin/users/user_a|Open the user> · " +
          "<https://us.posthog.com/replay/s|Watch the replay>",
      );
    });

    it("should log a post that Slack refuses", async () => {
      fetch.mockResolvedValue(new Response("invalid_token", { status: 403 }));
      await webhook({ url: URL, origin: ORIGIN }).post(NOTE);
      expect(error).toHaveBeenCalledWith("slack post failed", {
        title: "New account",
        status: 403,
        body: "invalid_token",
      });
    });

    it("should log a post that cannot reach Slack", async () => {
      const cause = new Error("offline");
      fetch.mockRejectedValue(cause);
      await webhook({ url: URL, origin: ORIGIN }).post(NOTE);
      expect(error).toHaveBeenCalledWith(
        "slack post failed",
        { title: "New account" },
        cause,
      );
    });
  });

  describe("announce", () => {
    afterEach(() => {
      vi.restoreAllMocks();
    });

    it("should post the note it makes", async () => {
      const slack = memory();
      await announce(slack, async () => NOTE);
      expect(slack.posted).toEqual([NOTE]);
    });

    it("should post nothing when there is no note", async () => {
      const slack = memory();
      await announce(slack, async () => null);
      expect(slack.posted).toEqual([]);
    });

    it("should log a note it cannot make instead of throwing", async () => {
      const error = vi.spyOn(console, "error").mockImplementation(() => {});
      const cause = new Error("Clerk is down");
      const slack = memory();
      await announce(slack, async () => {
        throw cause;
      });
      expect(slack.posted).toEqual([]);
      expect(error).toHaveBeenCalledWith("slack note failed", cause);
    });
  });

  describe("notes", () => {
    it("should announce a new account", () => {
      expect(joined(ADA)).toEqual({
        title: "New account",
        facts: [
          { label: "Name", value: "Ada Lovelace" },
          { label: "Email", value: "ada@example.com" },
        ],
        links: [{ label: "Open the user", href: "/admin/users/user_a" }],
      });
    });

    it("should announce a first Desktop login with the machine", () => {
      expect(firstLogin(ADA, "stand-1")).toEqual({
        title: "First Desktop login",
        facts: [
          { label: "Name", value: "Ada Lovelace" },
          { label: "Email", value: "ada@example.com" },
          { label: "Machine", value: "stand-1" },
        ],
        links: [{ label: "Open the user", href: "/admin/users/user_a" }],
      });
    });

    it("should announce a refused activation with its reason", () => {
      const acme: Organization = {
        key: LICENSE.organization,
        kind: "team",
        name: "Acme",
        clerkOrgID: "org_acme",
        ownerUserID: null,
        createdAt: NOW,
      };
      expect(
        refused({
          organization: acme,
          license: LICENSE,
          reason: "no_seats",
          actor: ADA,
        }),
      ).toEqual({
        title: "Activation refused",
        facts: [
          { label: "Organization", value: "Acme" },
          { label: "License", value: "Test rig" },
          { label: "Reason", value: "Every seat is taken" },
          { label: "By", value: "Ada Lovelace (ada@example.com)" },
        ],
        links: [{ label: "Open the license", href: `/licenses/${LICENSE.key}` }],
      });
    });

    it("should report a day of use", () => {
      const note = digest({
        day: new Date("2026-09-16T00:00:00Z"),
        active: { day: 2, week: 3, month: 4 },
        accounts: 5,
        firstLogins: 1,
      });
      expect(note.title).toBe("Synnax Desktop on 2026-09-16");
      expect(note.facts).toEqual([
        { label: "Daily active users", value: "2" },
        { label: "Weekly active users", value: "3" },
        { label: "Monthly active users", value: "4" },
        { label: "New accounts", value: "5" },
        { label: "First Desktop logins", value: "1" },
      ]);
      expect(note.links).toEqual([{ label: "Open the users", href: "/admin/users" }]);
    });

    describe("active", () => {
      const arrival = {
        account: { ...ADA, joinedAt: NOW },
        machine: "stand-1",
        at: new Date("2026-09-17T14:02:00Z"),
        days: 12,
      };

      it("should announce the first Desktop use of a day", () => {
        expect(
          active({ ...arrival, previous: new Date("2026-09-08T23:00:00Z") }),
        ).toEqual({
          title: "Ada Lovelace is active today",
          facts: [
            { label: "Email", value: "ada@example.com" },
            { label: "Machine", value: "stand-1" },
            { label: "Time", value: "14:02 UTC" },
            { label: "Last active", value: "9 days ago" },
            { label: "Active days", value: "12 of the last 30" },
          ],
          links: [{ label: "Open the user", href: "/admin/users/user_a" }],
        });
      });

      it("should say yesterday, or never, for the last use", () => {
        const lastActive = (previous: Date | null): string | undefined =>
          active({ ...arrival, previous }).facts.find((f) => f.label === "Last active")
            ?.value;
        expect(lastActive(new Date("2026-09-16T00:00:00Z"))).toBe("Yesterday");
        expect(lastActive(null)).toBe("Never");
      });
    });

    describe("recap", () => {
      const VISIT: Visit = {
        distinctID: "user_a",
        email: "ada@example.com",
        start: new Date("2026-09-17T14:02:00Z"),
        end: new Date("2026-09-17T15:40:30Z"),
        version: "1.0.0",
        os: "macos",
        screens: ["schematic", "lineplot"],
        actions: { channel_created: 2, schematic_created: 1, task_started: 1 },
        replay: "0192-session",
      };

      it("should report the time, the app, the actions, and the screens", () => {
        expect(recap(VISIT, { ...ADA, joinedAt: NOW })).toEqual({
          title: "Ada Lovelace used Synnax Desktop for 1 h 38 min",
          facts: [
            { label: "Email", value: "ada@example.com" },
            { label: "Time", value: "14:02 to 15:40 UTC" },
            { label: "App", value: "1.0.0 on macos" },
            {
              label: "Actions",
              value: "2 channels created, 1 task started, 1 schematic created",
            },
            { label: "Screens", value: "schematic, lineplot" },
          ],
          links: [
            { label: "Open the user", href: "/admin/users/user_a" },
            {
              label: "Watch the replay",
              href: "https://us.posthog.com/project/621680/replay/0192-session",
            },
          ],
        });
      });

      it("should report a session with no user, app, actions, or screens", () => {
        const note = recap(
          {
            ...VISIT,
            distinctID: "install-1",
            email: null,
            end: new Date("2026-09-17T14:02:40Z"),
            version: null,
            screens: [],
            actions: {},
          },
          null,
        );
        expect(note.title).toBe(
          "Someone who is not logged in used Synnax Desktop for under a minute",
        );
        expect(note.facts).toEqual([
          { label: "Time", value: "14:02 to 14:02 UTC" },
          { label: "Actions", value: "None" },
          { label: "Screens", value: "None" },
        ]);
        expect(note.links.map((l) => l.label)).toEqual(["Watch the replay"]);
      });

      it("should name a user Clerk does not know by their email", () => {
        const note = recap(VISIT, null);
        expect(note.title).toBe("ada@example.com used Synnax Desktop for 1 h 38 min");
        expect(note.facts[0]).toEqual({ label: "Time", value: "14:02 to 15:40 UTC" });
      });

      it("should read whole hours without minutes", () => {
        const end = new Date("2026-09-17T16:02:00Z");
        expect(recap({ ...VISIT, end }, null).title).toMatch(/for 2 h$/);
      });
    });
  });
});
