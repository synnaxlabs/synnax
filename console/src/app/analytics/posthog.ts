// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Analytics } from "@/platform/analytics";

/**
 * The Desktop project. The key is public, and every install carries it: it grants
 * nothing but writing events into that one project.
 */
const KEY = "phc_CaKeC8DVDWCYaiYSnUpEaWWXLrWptuXwgF7tTY3U6Q7t";

const API_HOST = "https://us.i.posthog.com";

/** Pinned, so a posthog release cannot change the behavior of a shipped install. */
const DEFAULTS = "2026-08-30";

/**
 * Properties posthog reads off the host page. The window title holds the project name,
 * and the rest describe a `tauri://` origin that means nothing. Screens carry a
 * synthesized address instead.
 */
const DENYLIST = [
  "$title",
  "$referrer",
  "$referring_domain",
  "$initial_referrer",
  "$initial_referring_domain",
  "$initial_current_url",
  "$initial_pathname",
  "$initial_host",
];

/** The group type that holds each machine and what it has built. */
const INSTALL_GROUP = "install";

export interface Params {
  /** Identifies the install across launches and across a webview data wipe. */
  installID: string;
}

const init = async (installID: string) => {
  const [{ default: posthog }] = await Promise.all([
    import("posthog-js/dist/module.no-external"),
    // Static bundles, because remote script loading stays off. Both must load before
    // `posthog.init`; this bundle carries no loader to fetch them later. The recorder
    // is left out entirely, so no session replay code ships.
    import("posthog-js/dist/surveys"),
    import("posthog-js/dist/exception-autocapture"),
  ]);
  posthog.init(KEY, {
    api_host: API_HOST,
    defaults: DEFAULTS,
    // Logs every event to the console of the window, so a dev run can be read without
    // waiting for the project to ingest it.
    debug: IS_DEV,
    disable_external_dependency_loading: true,
    persistence: "localStorage",
    // The install starts anonymous. Signing in identifies the account, which merges the
    // events before it into the account's person.
    bootstrap: { distinctID: installID },
    // Element text is channel, device, and schematic names, so autocapture keeps the
    // shape of an interaction and drops every string in it.
    autocapture: true,
    mask_all_text: true,
    mask_all_element_attributes: true,
    // Screens are synthesized from the focused tab; there is no navigation to observe.
    capture_pageview: false,
    capture_pageleave: false,
    disable_session_recording: true,
    capture_exceptions: true,
    capture_heatmaps: false,
    capture_dead_clicks: false,
    capture_performance: false,
    property_denylist: DENYLIST,
    // An exception message names the channel or the device that failed, so only the
    // type and the stack leave the machine.
    before_send: (event) => {
      if (event == null || event.event !== "$exception") return event;
      delete event.properties.$exception_message;
      const list: unknown = event.properties.$exception_list;
      if (Array.isArray(list))
        list.forEach((exc) => {
          if (exc != null && typeof exc === "object") delete exc.value;
        });
      return event;
    },
  });
  attach(posthog, installID);
  return posthog;
};

type PostHog = Awaited<ReturnType<typeof init>>;

/** Tags every later event with the environment and the install. A reset clears both. */
const attach = (posthog: PostHog, installID: string): void => {
  // A dev run reports as an ordinary install otherwise, so every count needs this to
  // separate the two.
  posthog.register({ environment: IS_DEV ? "development" : "production" });
  posthog.group(INSTALL_GROUP, installID);
};

interface Loaded {
  posthog: PostHog;
  installID: string;
}

/**
 * Returns a transport over posthog right away and loads posthog behind it. Calls made
 * while it loads run once it is there, so the first seconds of a launch report like
 * any other. The import is dynamic, so the Console build, where this module is
 * unreachable, carries no analytics code at all.
 */
export const create = (params: Promise<Params>): Analytics.Transport => {
  let loaded: Loaded | null = null;
  let failed = false;
  let waiting: Array<(loaded: Loaded) => void> = [];
  const load = async (): Promise<void> => {
    try {
      const { installID } = await params;
      const next = { posthog: await init(installID), installID };
      loaded = next;
      waiting.forEach((run) => run(next));
    } catch (err) {
      failed = true;
      console.error("failed to load analytics", err);
    } finally {
      waiting = [];
    }
  };
  void load();
  const run = (call: (loaded: Loaded) => void): void => {
    if (loaded != null) call(loaded);
    else if (!failed) waiting.push(call);
  };
  return {
    capture: (event, properties) =>
      run(({ posthog }) => posthog.capture(event, properties)),
    identify: ({ id, email }) => run(({ posthog }) => posthog.identify(id, { email })),
    reset: () =>
      run(({ posthog, installID }) => {
        posthog.reset();
        attach(posthog, installID);
      }),
    describe: (properties) =>
      run(({ posthog, installID }) =>
        posthog.group(INSTALL_GROUP, installID, properties),
      ),
  };
};
