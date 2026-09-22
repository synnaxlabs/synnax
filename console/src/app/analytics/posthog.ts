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
 * The project the docs site already reports to. Desktop shares it so that a later
 * Portal sign-in can join a website visit to an install; two projects never merge.
 */
const KEY = "phc_NX1V2suy6Rd924qT3hjDCM23miC3SxqoP7r1GHF8Vsq";

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

export interface Params {
  /** Identifies the install across launches and across a webview data wipe. */
  installID: string;
}

const init = async (params: Promise<Params>) => {
  const [{ installID }, { default: posthog }] = await Promise.all([
    params,
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
    disable_external_dependency_loading: true,
    persistence: "localStorage",
    // The bootstrap identifies the install, so the default `identified_only` still
    // earns person properties. Forcing profiles on would only buy a paying profile per
    // random id whenever the install id is missing.
    bootstrap: { distinctID: installID, isIdentifiedID: true },
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
  posthog.register({ app: "desktop" });
  return posthog;
};

type PostHog = Awaited<ReturnType<typeof init>>;

/**
 * Returns a transport over posthog right away and loads posthog behind it. Calls made
 * while it loads run once it is there, so the first seconds of a launch report like
 * any other. The import is dynamic, so the Console build, where this module is
 * unreachable, carries no analytics code at all.
 */
export const create = (params: Promise<Params>): Analytics.Transport => {
  let loaded: PostHog | null = null;
  let failed = false;
  let waiting: Array<(posthog: PostHog) => void> = [];
  const load = async (): Promise<void> => {
    try {
      const posthog = await init(params);
      loaded = posthog;
      waiting.forEach((run) => run(posthog));
    } catch (err) {
      failed = true;
      console.error("failed to load analytics", err);
    } finally {
      waiting = [];
    }
  };
  void load();
  const run = (call: (posthog: PostHog) => void): void => {
    if (loaded != null) call(loaded);
    else if (!failed) waiting.push(call);
  };
  return {
    capture: (event, properties) => run((p) => p.capture(event, properties)),
    describe: (properties) => run((p) => p.setPersonProperties(properties)),
  };
};
