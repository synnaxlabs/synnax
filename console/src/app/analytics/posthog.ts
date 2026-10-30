// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Analytics } from "@/platform/analytics";

/**
 * The Synnax project, which the websites share. The key is public, and every install
 * carries it: it grants nothing but writing events into that one project.
 */
const KEY = "phc_CaKeC8DVDWCYaiYSnUpEaWWXLrWptuXwgF7tTY3U6Q7t";

/** Our own address for PostHog, which ad blockers do not know. */
const API_HOST = "https://t.synnaxlabs.com";

/** Pinned, so a PostHog release cannot change the behavior of a shipped install. */
const DEFAULTS = "2026-08-30";

/**
 * Properties PostHog reads off the host page. The window title holds the project name,
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

/**
 * Monaco rejects a task it cancels with an error whose name and message are both
 * `Canceled`, and nothing awaits the rejection. It is not a failure, and VS Code drops
 * it too.
 */
const monacoCancellationZ = z
  .array(z.object({ type: z.literal("Canceled"), value: z.literal("Canceled") }))
  .nonempty();

const isMonacoCancellation = (exceptions: unknown): boolean =>
  monacoCancellationZ.safeParse(exceptions).success;

export interface Params {
  /** Identifies the install across launches and across a webview data wipe. */
  installID: string;
}

/**
 * The PostHog session that every window of a launch reports in. PostHog reads another
 * window's session from storage only at init and after an idle time, so windows would
 * otherwise each move to sessions of their own.
 */
export interface SharedSession {
  /** Resolves to the session once the main window has started one. */
  join: () => Promise<string>;
  /** Returns the session that the windows share now. */
  current: () => string | undefined;
  /** Shares a session that PostHog moved this window to, such as after an idle time. */
  share: (id: string) => void;
  /** Calls the listener on each change to the shared session, until unsubscribed. */
  subscribe: (listener: (id: string) => void) => () => void;
}

const importPostHog = async () => {
  const [{ default: posthog }] = await Promise.all([
    import("posthog-js/dist/module.no-external"),
    // Static bundles, because remote script loading stays off. All must load before
    // `posthog.init`; this bundle carries no loader to fetch them later.
    import("posthog-js/dist/surveys"),
    import("posthog-js/dist/exception-autocapture"),
    import("posthog-js/dist/posthog-recorder"),
  ]);
  return posthog;
};

type PostHog = Awaited<ReturnType<typeof importPostHog>>;

/** Tags every later event with the environment and the install. A reset clears both. */
const attach = (posthog: PostHog, installID: string): void => {
  // Local and test builds otherwise report as ordinary installs, so every count needs
  // this to leave them out.
  posthog.register({
    environment: RELEASE ? "production" : "development",
    install_id: installID,
  });
};

/** Captures again after a stop. PostHog stores the opt-out, so it outlives a launch. */
const resume = (posthog: PostHog, installID: string): void => {
  if (posthog.has_opted_out_capturing())
    posthog.opt_in_capturing({ captureEventName: false });
  attach(posthog, installID);
};

const init = (posthog: PostHog, installID: string, sessionID: string): void => {
  posthog.init(KEY, {
    api_host: API_HOST,
    ui_host: "https://us.posthog.com",
    defaults: DEFAULTS,
    // The defaults mark a `localhost` host as a test user, and macOS serves the app
    // from `tauri://localhost`.
    internal_or_test_user_hostname: null,
    // Logs every event to the console of the window, so a dev run can be read without
    // waiting for the project to ingest it.
    debug: IS_DEV,
    disable_external_dependency_loading: true,
    persistence: "localStorage",
    // The install starts anonymous. Signing in identifies the account, which merges the
    // events before it into the account's person.
    bootstrap: { distinctID: installID, sessionID },
    // Element text is channel, device, and schematic names, so autocapture keeps the
    // shape of an interaction and drops every string in it.
    autocapture: true,
    mask_all_text: true,
    mask_all_element_attributes: true,
    // Screens are synthesized from the focused tab; there is no navigation to observe.
    capture_pageview: false,
    capture_pageleave: false,
    // Replays show the text of the app and what the user types, except passwords.
    // Plots, logs, and values draw on canvases in a worker, which replays show blank.
    session_recording: { maskAllInputs: false },
    capture_exceptions: true,
    capture_heatmaps: false,
    capture_dead_clicks: false,
    capture_performance: false,
    property_denylist: DENYLIST,
    // Exception messages go out unchanged, since a crash report needs them. They can
    // name a channel or a device.
    before_send: (event) =>
      event?.event === "$exception" &&
      isMonacoCancellation(event.properties.$exception_list)
        ? null
        : event,
  });
  resume(posthog, installID);
};

interface Loaded {
  posthog: PostHog;
  installID: string;
}

/**
 * Makes PostHog forget the account. The session stays the shared one: a plain reset
 * goes back to the session this window joined at init.
 */
const forget = ({ posthog, installID }: Loaded, session: SharedSession): void =>
  posthog.reset({ bootstrap: { distinctID: installID, sessionID: session.current() } });

/** A transport that loads nothing and sends nothing until it starts. */
export interface Transport extends Analytics.Transport {
  /** Loads PostHog on the first call, then sends every call until {@link stop}. */
  start: () => void;
  /** Drops every later call, then resets PostHog and opts it out of all requests. */
  stop: () => void;
}

/**
 * Returns a transport over PostHog right away. PostHog loads on the first start, and
 * calls made while it loads run once it is there, so the first seconds of a launch
 * report like any other. The import is dynamic, so the Console build, where this module
 * is unreachable, carries no analytics code at all.
 */
export const create = (params: Promise<Params>, session: SharedSession): Transport => {
  let started = false;
  let loading = false;
  let failed = false;
  let loaded: Loaded | null = null;
  let waiting: Array<(loaded: Loaded) => void> = [];
  const load = async (): Promise<void> => {
    loading = true;
    try {
      const [{ installID }, posthog] = await Promise.all([params, importPostHog()]);
      const sessionID = await session.join();
      // PostHog sends from init on, and the user can refuse while the bundles load.
      if (started) {
        init(posthog, installID, sessionID);
        posthog.onSessionId(session.share);
        // Setting the session calls no onSessionId listener, so a move made in another
        // window is not shared back.
        session.subscribe((id) => {
          if (id !== posthog.get_session_id())
            posthog.sessionManager?.setBootstrapSessionId(id);
        });
        const next = { posthog, installID };
        loaded = next;
        waiting.forEach((run) => run(next));
      }
    } catch (err) {
      failed = true;
      console.error("failed to load analytics", err);
    } finally {
      loading = false;
      waiting = [];
    }
  };
  const run = (call: (loaded: Loaded) => void): void => {
    if (!started) return;
    if (loaded != null) call(loaded);
    else if (!failed) waiting.push(call);
  };
  return {
    start: () => {
      if (started) return;
      started = true;
      if (loaded == null) {
        if (!loading && !failed) void load();
        return;
      }
      loaded.posthog.set_config({ advanced_disable_flags: false });
      resume(loaded.posthog, loaded.installID);
    },
    stop: () => {
      if (!started) return;
      started = false;
      waiting = [];
      if (loaded == null) return;
      const { posthog } = loaded;
      // A reset reloads feature flags, which PostHog requests even while opted out.
      posthog.set_config({ advanced_disable_flags: true });
      // A reset clears the opt-out that PostHog stores, so the reset goes first.
      forget(loaded, session);
      posthog.opt_out_capturing();
    },
    capture: (event, properties) =>
      run(({ posthog }) => posthog.capture(event, properties)),
    identify: ({ id, email }) => run(({ posthog }) => posthog.identify(id, { email })),
    reset: () =>
      run((loaded) => {
        forget(loaded, session);
        attach(loaded.posthog, loaded.installID);
      }),
  };
};
