// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { capture, type PostHog, reset, start } from "./analytics";
import Analytics from "./Analytics.astro";

type Init = Parameters<PostHog["init"]>[1];

const render = async (props: Record<string, unknown>): Promise<string> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  return await container.renderToString(Analytics, { props });
};

/** Records what the sites ask of PostHog, the way the loader queues it. */
const recorder = (consent: "granted" | "denied" | "pending" = "pending") => {
  const calls: string[] = [];
  let config: Init | null = null;
  const posthog: PostHog = {
    init: (_, c) => {
      config = c;
      calls.push("init");
    },
    capture: (event) => calls.push(`capture ${event}`),
    identify: (id) => calls.push(`identify ${id}`),
    opt_in_capturing: (options) =>
      calls.push(options == null ? "opt_in" : "opt_in silent"),
    opt_out_capturing: () => calls.push("opt_out"),
    reset: () => calls.push("reset"),
    get_explicit_consent_status: () => consent,
  };
  const configured = (): Init => {
    if (config == null) throw new Error("PostHog was never initialized");
    return config;
  };
  return { posthog, calls, configured };
};

const answered = (): void => {
  document.cookie = "synnax_consent=1";
};

const country = (mode: string) => vi.fn(async () => new Response(mode));

const click = (selector: string): void =>
  (document.querySelector(selector) as HTMLElement).click();

const card = (): HTMLElement => document.querySelector(".consent") as HTMLElement;

const begin = async ({
  mode = "ask",
  userID = null,
  textMasked = false,
  consent,
}: {
  mode?: string;
  userID?: string | null;
  textMasked?: boolean;
  consent?: "granted" | "denied" | "pending";
} = {}) => {
  const rec = recorder(consent);
  const fetch = country(mode);
  await start({ posthog: () => rec.posthog, fetch, card: card(), userID, textMasked });
  return { ...rec, fetch };
};

describe("analytics", () => {
  describe("server render", () => {
    const rendered = async (props: Record<string, unknown>): Promise<HTMLElement> => {
      const { document } = new JSDOM(await render(props)).window;
      const el = document.querySelector<HTMLElement>(".consent");
      if (el == null) throw new Error("the consent card did not render");
      return el;
    };

    it("should hide the card until the script decides to show it", async () => {
      expect((await rendered({ privacy: "/privacy" })).hidden).toBe(true);
    });

    it("should link the card to the privacy policy", async () => {
      const el = await rendered({ privacy: "/privacy-policy" });
      expect(el.querySelector("a")?.getAttribute("href")).toBe("/privacy-policy");
    });

    it("should hand the account and the text masking to the script", async () => {
      const el = await rendered({
        privacy: "/privacy",
        userID: "user_1",
        textMasked: true,
      });
      expect(el.dataset.userId).toBe("user_1");
      expect(el.dataset.textMasked).toBe("");
    });

    it("should hand neither to the script by default", async () => {
      const el = await rendered({ privacy: "/privacy" });
      expect(el.dataset.userId).toBeUndefined();
      expect(el.dataset.textMasked).toBeUndefined();
    });
  });

  describe("browser", () => {
    let markup: string;

    beforeAll(async () => {
      // The footer holds the settings button.
      markup = `${await render({ privacy: "/privacy" })}
        <button class="settings" data-consent="settings"></button>`;
    });

    beforeEach(() => {
      const { window } = new JSDOM(markup, { url: "https://www.synnaxlabs.com/" });
      vi.stubGlobal("document", window.document);
      vi.stubGlobal("Element", window.Element);
    });

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    describe("a visitor in Europe who has not answered", () => {
      it("should count them without cookies and show the card", async () => {
        const { calls, configured } = await begin({ mode: "ask" });
        expect(calls).toEqual(["init"]);
        expect(configured().cookieless_mode).toBe("on_reject");
        expect(configured().opt_out_capturing_by_default).toBe(true);
        expect(card().hidden).toBe(false);
      });

      it("should turn cookies on and link the account when they accept", async () => {
        const { calls } = await begin({ mode: "ask", userID: "user_1" });
        click("[data-consent=accept]");
        expect(calls).toEqual(["init", "opt_in", "identify user_1"]);
        expect(card().hidden).toBe(true);
      });

      it("should keep counting without cookies when they reject", async () => {
        const { calls } = await begin({ mode: "ask", userID: "user_1" });
        click("[data-consent=reject]");
        expect(calls).toEqual(["init", "opt_out"]);
        expect(card().hidden).toBe(true);
      });
    });

    describe("a visitor elsewhere who has not answered", () => {
      it("should turn cookies on without showing the card", async () => {
        const { calls, configured } = await begin({ mode: "notice" });
        expect(calls).toEqual(["init", "opt_in silent"]);
        expect(configured().opt_out_capturing_by_default).toBe(false);
        expect(card().hidden).toBe(true);
      });

      it("should link the account", async () => {
        const { calls } = await begin({ mode: "notice", userID: "user_1" });
        expect(calls).toEqual(["init", "opt_in silent", "identify user_1"]);
      });
    });

    describe("a visitor who has answered", () => {
      it("should keep their answer without asking the country", async () => {
        answered();
        const { calls, fetch } = await begin();
        expect(fetch).not.toHaveBeenCalled();
        expect(calls).toEqual(["init"]);
        expect(card().hidden).toBe(true);
      });

      it("should link the account once PostHog loads with cookies on", async () => {
        answered();
        const { posthog, calls, configured } = await begin({
          userID: "user_1",
          consent: "granted",
        });
        configured().loaded(posthog);
        expect(calls).toEqual(["init", "identify user_1"]);
      });

      it("should not link the account when they rejected", async () => {
        answered();
        const { posthog, calls, configured } = await begin({
          userID: "user_1",
          consent: "denied",
        });
        configured().loaded(posthog);
        expect(calls).toEqual(["init"]);
      });
    });

    it("should open the card from the cookie settings", async () => {
      answered();
      await begin();
      click(".settings");
      expect(card().hidden).toBe(false);
    });

    it("should send to our own address and share the answer in a cookie", async () => {
      const { configured } = await begin();
      expect(configured().api_host).toBe("https://t.synnaxlabs.com");
      expect(configured().consent_persistence_name).toBe("synnax_consent");
      expect(configured().opt_out_capturing_persistence_type).toBe("cookie");
    });

    it("should hide all text in recordings only when asked", async () => {
      expect((await begin()).configured().session_recording).toEqual({});
      expect(
        (await begin({ textMasked: true })).configured().session_recording,
      ).toEqual({ maskTextSelector: "*" });
    });

    it("should reject an unknown mode from the country route", async () => {
      await expect(begin({ mode: "maybe" })).rejects.toThrow(
        "unknown consent mode: maybe",
      );
    });
  });

  describe("calls", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("should send an event that survives a click leaving the page", () => {
      const sent = vi.fn();
      vi.stubGlobal("window", { posthog: { capture: sent } });
      capture("desktop_download_clicked", { location: "hero" });
      expect(sent).toHaveBeenCalledExactlyOnceWith(
        "desktop_download_clicked",
        { location: "hero" },
        { transport: "sendBeacon" },
      );
    });

    it("should make PostHog forget the visitor", () => {
      const forgot = vi.fn();
      vi.stubGlobal("window", { posthog: { reset: forgot } });
      reset();
      expect(forgot).toHaveBeenCalledOnce();
    });
  });
});
