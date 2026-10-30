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
import { type DOMWindow, JSDOM } from "jsdom";
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
  sensitive = false,
  consent,
}: {
  mode?: string;
  userID?: string | null;
  sensitive?: boolean;
  consent?: "granted" | "denied" | "pending";
} = {}) => {
  const rec = recorder(consent);
  const fetch = country(mode);
  await start({ posthog: () => rec.posthog, fetch, card: card(), userID, sensitive });
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

    it("should hand the account and the sensitivity to the script", async () => {
      const el = await rendered({
        privacy: "/privacy",
        userID: "user_1",
        sensitive: true,
      });
      expect(el.dataset.userId).toBe("user_1");
      expect(el.dataset.sensitive).toBe("");
    });

    it("should hand neither to the script by default", async () => {
      const el = await rendered({ privacy: "/privacy" });
      expect(el.dataset.userId).toBeUndefined();
      expect(el.dataset.sensitive).toBeUndefined();
    });
  });

  describe("loader", () => {
    /** Runs the loader on the given queue, which stands in for `window.posthog`. */
    const load = async (queue: unknown[]): Promise<DOMWindow> =>
      new JSDOM(await render({ privacy: "/privacy" }), {
        runScripts: "dangerously",
        beforeParse: (window) => Object.assign(window, { posthog: queue }),
      }).window;

    it("should queue calls made before PostHog starts", async () => {
      const queue: unknown[] = [];
      const window = await load(queue);
      window.posthog.capture("desktop_download_clicked", { location: "hero" });
      window.posthog.reset();
      expect([...queue]).toEqual([
        ["capture", "desktop_download_clicked", { location: "hero" }],
        ["reset"],
      ]);
    });

    it("should load PostHog from the API host once", async () => {
      const queue: unknown[] = [];
      const window = await load(queue);
      const config = { api_host: "https://t.synnaxlabs.com" } as Init;
      window.posthog.init("key", config);
      window.posthog.init("key", config);
      const scripts = window.document.querySelectorAll(
        'script[src="https://t.synnaxlabs.com/static/array.js"]',
      );
      expect(scripts).toHaveLength(1);
      expect([...queue]).toEqual([["init", "key", config]]);
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

    describe("a visitor who answers while the country loads", () => {
      /** Starts with a country route that answers only when told to. */
      const beginWaiting = () => {
        const rec = recorder();
        let respond: (mode: string) => void = () => {};
        const fetch = vi.fn(
          async () =>
            await new Promise<Response>((resolve) => {
              respond = (mode) => resolve(new Response(mode));
            }),
        );
        const started = start({
          posthog: () => rec.posthog,
          fetch,
          card: card(),
          userID: null,
          sensitive: false,
        });
        return { ...rec, started, respond: (mode: string) => respond(mode) };
      };

      it("should keep a rejection outside Europe", async () => {
        const { calls, started, respond } = beginWaiting();
        click(".settings");
        click("[data-consent=reject]");
        respond("notice");
        await started;
        expect(calls).toEqual(["opt_out", "init"]);
      });

      it("should not show the card again in Europe", async () => {
        const { started, respond } = beginWaiting();
        click(".settings");
        click("[data-consent=accept]");
        respond("ask");
        await started;
        expect(card().hidden).toBe(true);
      });
    });

    describe("a visitor who logs out before PostHog loads", () => {
      /** Resets through the loader's queue, which dies with the page. */
      const logOut = (): string[] => {
        const queue = recorder();
        vi.stubGlobal("window", { posthog: queue.posthog });
        reset();
        return queue.calls;
      };

      it("should make PostHog forget them on the next page that loads it", async () => {
        expect(logOut()).toEqual([]);
        answered();
        const { posthog, calls, configured } = await begin();
        configured().loaded(posthog);
        expect(calls).toEqual(["init", "reset"]);
      });

      it("should forget them before it links an account", async () => {
        logOut();
        answered();
        const { posthog, calls, configured } = await begin({
          userID: "user_1",
          consent: "granted",
        });
        configured().loaded(posthog);
        expect(calls).toEqual(["init", "reset", "identify user_1"]);
      });

      it("should forget them only once", async () => {
        logOut();
        answered();
        const first = await begin();
        first.configured().loaded(first.posthog);
        const second = await begin();
        second.configured().loaded(second.posthog);
        expect(second.calls).toEqual(["init"]);
      });

      it("should forget them on a host outside synnaxlabs.com", async () => {
        const { window } = new JSDOM(markup, { url: "http://localhost:4321/" });
        vi.stubGlobal("document", window.document);
        logOut();
        answered();
        const { posthog, calls, configured } = await begin();
        configured().loaded(posthog);
        expect(calls).toEqual(["init", "reset"]);
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

    describe("URLs", () => {
      const handoff = "https://portal.synnaxlabs.com/licenses/activate?handoff=a#done";
      const bare = "https://portal.synnaxlabs.com/licenses/activate";

      it("should cut the query and fragment off each URL in an event", async () => {
        const { configured } = await begin();
        const event = configured().before_send({
          properties: {
            $current_url: handoff,
            $web_vitals_LCP_event: { $current_url: handoff },
            $el_text: "Why?",
            $screen_height: 900,
          },
          $set_once: { $initial_current_url: handoff },
        });
        expect(event).toEqual({
          properties: {
            $current_url: bare,
            $web_vitals_LCP_event: { $current_url: bare },
            $el_text: "Why?",
            $screen_height: 900,
          },
          $set_once: { $initial_current_url: bare },
        });
      });

      it("should merge heatmap points of URLs that differ only by query", async () => {
        const { configured } = await begin();
        const event = configured().before_send({
          properties: {
            $heatmap_data: { [`${bare}?handoff=a`]: [{ x: 1 }], [handoff]: [{ x: 2 }] },
          },
        });
        expect(event.properties).toEqual({
          $heatmap_data: { [bare]: [{ x: 1 }, { x: 2 }] },
        });
      });

      it("should cut the query and fragment off each URL in recordings", async () => {
        const { configured } = await begin();
        const mask = configured().session_recording.maskCapturedNetworkRequestFn;
        expect(mask({ name: handoff, method: "GET" })).toEqual({
          name: bare,
          method: "GET",
        });
      });
    });

    describe("a sensitive page", () => {
      it("should keep page text and attributes out of PostHog", async () => {
        const { configured } = await begin({ sensitive: true });
        expect(configured().mask_all_text).toBe(true);
        expect(configured().mask_all_element_attributes).toBe(true);
        expect(configured().session_recording.maskTextSelector).toBe("*");
      });

      it("should cut the query off links in recordings", async () => {
        const { configured } = await begin({ sensitive: true });
        const mask = configured().session_recording.maskAttributeFn;
        expect(mask?.("href", "/login?redirect_url=a")).toBe("/login");
        expect(mask?.("src", "/logo.png?w=2")).toBe("/logo.png?w=2");
      });
    });

    it("should keep page text and attributes on other pages", async () => {
      const { configured } = await begin();
      expect(configured().mask_all_text).toBe(false);
      expect(configured().mask_all_element_attributes).toBe(false);
      expect(configured().session_recording.maskTextSelector).toBeUndefined();
      expect(configured().session_recording.maskAttributeFn).toBeUndefined();
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

    it("should make PostHog forget the visitor once it has loaded", () => {
      const forgot = vi.fn();
      vi.stubGlobal("window", { posthog: { __loaded: true, reset: forgot } });
      reset();
      expect(forgot).toHaveBeenCalledOnce();
    });
  });
});
