// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mode } from "./consent";

/** The Synnax project, which Desktop shares. */
const KEY = "phc_CaKeC8DVDWCYaiYSnUpEaWWXLrWptuXwgF7tTY3U6Q7t";

/** Our own address for PostHog, which ad blockers do not know. */
const API_HOST = "https://t.synnaxlabs.com";

/** The visitor's answer, in a cookie that every synnaxlabs.com site shares. */
const CHOICE = "synnax_consent";

/** The parts of PostHog that the sites call. */
export interface PostHog {
  init: (key: string, config: Config) => void;
  capture: (
    event: string,
    properties?: Record<string, string>,
    options?: { transport: "sendBeacon" },
  ) => void;
  identify: (id: string) => void;
  opt_in_capturing: (options?: { captureEventName: false }) => void;
  opt_out_capturing: () => void;
  reset: () => void;
  get_explicit_consent_status: () => "granted" | "denied" | "pending";
}

interface Config {
  api_host: string;
  ui_host: string;
  defaults: string;
  person_profiles: "identified_only";
  cookieless_mode: "on_reject";
  opt_out_capturing_by_default: boolean;
  consent_persistence_name: string;
  opt_out_capturing_persistence_type: "cookie";
  session_recording: { maskTextSelector?: string };
  loaded: (posthog: PostHog) => void;
}

declare global {
  interface Window {
    /**
     * Defined by the snippet in `Analytics.astro` before any page script runs. It
     * queues calls until PostHog loads, then becomes PostHog, so read it at each call.
     */
    posthog: PostHog;
  }
}

export interface Options {
  posthog: () => PostHog;
  fetch: typeof fetch;
  /** The card that asks for consent. */
  card: HTMLElement;
  /** The account to link the visitor to once cookies are on. */
  userID: string | null;
  /** Hides all text in recordings, for pages that show other people's data. */
  textMasked: boolean;
}

const chosen = (): boolean =>
  document.cookie.split("; ").some((c) => c.startsWith(`${CHOICE}=`));

const fetchMode = async (fetch: Options["fetch"]): Promise<Mode> => {
  const mode = await (await fetch("/consent")).text();
  if (mode !== "ask" && mode !== "notice")
    throw new Error(`unknown consent mode: ${mode}`);
  return mode;
};

/**
 * Starts PostHog. A visitor who has not answered sees the card in Europe, and gets
 * cookies everywhere else. Until they answer, and after they reject, PostHog counts
 * them without cookies. A `[data-consent]` element accepts, rejects, or opens the card.
 */
export const start = async ({
  posthog,
  fetch,
  card,
  userID,
  textMasked,
}: Options): Promise<void> => {
  const identify = (to: PostHog): void => {
    if (userID != null) to.identify(userID);
  };
  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const action = e.target.closest<HTMLElement>("[data-consent]")?.dataset.consent;
    if (action == null) return;
    if (action === "settings") card.hidden = false;
    else if (action === "accept") {
      posthog().opt_in_capturing();
      identify(posthog());
      card.hidden = true;
    } else if (action === "reject") {
      posthog().opt_out_capturing();
      card.hidden = true;
    } else throw new Error(`unknown consent action: ${action}`);
  });
  const init = (notice: boolean): void =>
    posthog().init(KEY, {
      api_host: API_HOST,
      ui_host: "https://us.posthog.com",
      // Pinned, so a PostHog release cannot change what the sites record. This date
      // counts a page view on each client-side navigation in the docs.
      defaults: "2026-08-30",
      person_profiles: "identified_only",
      cookieless_mode: "on_reject",
      opt_out_capturing_by_default: !notice,
      consent_persistence_name: CHOICE,
      opt_out_capturing_persistence_type: "cookie",
      session_recording: textMasked ? { maskTextSelector: "*" } : {},
      loaded: (loaded) => {
        if (loaded.get_explicit_consent_status() === "granted") identify(loaded);
      },
    });
  if (chosen()) {
    init(false);
    return;
  }
  const mode = await fetchMode(fetch);
  init(mode === "notice");
  if (mode === "ask") card.hidden = false;
  else {
    posthog().opt_in_capturing({ captureEventName: false });
    identify(posthog());
  }
};
