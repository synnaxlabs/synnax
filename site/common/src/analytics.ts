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

/**
 * A reset that PostHog has not run yet, in a cookie that every synnaxlabs.com site
 * shares. A call queued before PostHog loads is lost when the page leaves.
 */
const PENDING_RESET = "synnax_reset";

/** As long as PostHog keeps the visitor's identity. */
const YEAR_SECONDS = 365 * 24 * 60 * 60;

const SITES = "synnaxlabs.com";

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
  /** True once PostHog has loaded. The loader's queue has no such field. */
  __loaded?: boolean;
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
  mask_all_text: boolean;
  mask_all_element_attributes: boolean;
  before_send: (event: CaptureResult) => CaptureResult;
  session_recording: Recording;
  loaded: (posthog: PostHog) => void;
}

type Properties = Record<string, unknown>;

/** The parts of an event that can hold URLs. */
interface CaptureResult {
  properties: Properties;
  $set?: Properties;
  $set_once?: Properties;
}

interface Recording {
  maskTextSelector?: string;
  maskAttributeFn?: (name: string, value: string) => string;
  maskCapturedNetworkRequestFn: <R extends { name: string }>(request: R) => R;
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

const hasCookie = (name: string): boolean =>
  document.cookie.split("; ").some((c) => c.startsWith(`${name}=`));

/** Sets a cookie for every synnaxlabs.com site, or for this host elsewhere. */
const setCookie = (name: string, maxAge: number): void => {
  const { hostname } = document.location;
  const shared = hostname === SITES || hostname.endsWith(`.${SITES}`);
  const domain = shared ? `; domain=${SITES}` : "";
  document.cookie = `${name}=1; path=/; max-age=${maxAge}; samesite=lax${domain}`;
};

/** Sends an event to PostHog. It survives a click that leaves the page. */
export const capture = (event: string, properties?: Record<string, string>): void =>
  window.posthog.capture(event, properties, { transport: "sendBeacon" });

/**
 * Makes PostHog forget the visitor, so later visits in this browser start anonymous.
 * Before PostHog loads, the next page that loads it forgets them instead.
 */
export const reset = (): void => {
  if (window.posthog.__loaded === true) window.posthog.reset();
  else setCookie(PENDING_RESET, YEAR_SECONDS);
};

export interface Options {
  posthog: () => PostHog;
  fetch: typeof fetch;
  /** The card that asks for consent. */
  card: HTMLElement;
  /** The account to link the visitor to once cookies are on. */
  userID: string | null;
  /**
   * For pages that show other people's data or put one-time codes in links. PostHog
   * then records no page text, no attributes of clicked elements, and no link queries.
   */
  sensitive: boolean;
}

/** Cuts the query and fragment off a URL. Portal links carry one-time codes there. */
const bare = (url: string): string => url.replace(/[?#].*/s, "");

const bareIfAbsolute = (value: string): string =>
  /^https?:\/\//.test(value) ? bare(value) : value;

const isRecord = (value: unknown): value is Properties =>
  typeof value === "object" && value != null && value.constructor === Object;

const scrub = (value: unknown): unknown => {
  if (typeof value === "string") return bareIfAbsolute(value);
  if (Array.isArray(value)) return value.map(scrub);
  if (isRecord(value)) return scrubProperties(value);
  return value;
};

/** Cuts the query and fragment off each URL in the properties, keys included. */
const scrubProperties = (properties: Properties): Properties => {
  const scrubbed: Properties = {};
  for (const [key, value] of Object.entries(properties)) {
    const k = bareIfAbsolute(key);
    const v = scrub(value);
    const prior = scrubbed[k];
    // Heatmaps key their points by page URL, so two keys can become one.
    scrubbed[k] = Array.isArray(prior) && Array.isArray(v) ? [...prior, ...v] : v;
  }
  return scrubbed;
};

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
 * PostHog never receives the query or fragment of a page URL.
 */
export const start = async ({
  posthog,
  fetch,
  card,
  userID,
  sensitive,
}: Options): Promise<void> => {
  const identify = (to: PostHog): void => {
    if (userID != null) to.identify(userID);
  };
  let answered = hasCookie(CHOICE);
  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const action = e.target.closest<HTMLElement>("[data-consent]")?.dataset.consent;
    if (action == null) return;
    if (action === "settings") card.hidden = false;
    else if (action === "accept") {
      answered = true;
      posthog().opt_in_capturing();
      identify(posthog());
      card.hidden = true;
    } else if (action === "reject") {
      answered = true;
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
      mask_all_text: sensitive,
      mask_all_element_attributes: sensitive,
      before_send: (event) => {
        event.properties = scrubProperties(event.properties);
        if (event.$set != null) event.$set = scrubProperties(event.$set);
        if (event.$set_once != null) event.$set_once = scrubProperties(event.$set_once);
        return event;
      },
      session_recording: {
        maskCapturedNetworkRequestFn: (request) => ({
          ...request,
          name: bare(request.name),
        }),
        ...(sensitive && {
          maskTextSelector: "*",
          maskAttributeFn: (name, value) => (name === "href" ? bare(value) : value),
        }),
      },
      loaded: (loaded) => {
        if (hasCookie(PENDING_RESET)) {
          loaded.reset();
          setCookie(PENDING_RESET, 0);
        }
        if (loaded.get_explicit_consent_status() === "granted") identify(loaded);
      },
    });
  const mode = answered ? null : await fetchMode(fetch);
  // The visitor can answer from the cookie settings while the country loads.
  if (answered || mode == null) {
    init(false);
    return;
  }
  init(mode === "notice");
  if (mode === "ask") card.hidden = false;
  else {
    posthog().opt_in_capturing({ captureEventName: false });
    identify(posthog());
  }
};
