// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { getVersion } from "@tauri-apps/api/app";

import { type Install, retrieveInstall } from "@/app/analytics/install";
import { create, type Params } from "@/app/analytics/posthog";
import { Analytics } from "@/platform/analytics";
import { Session } from "@/session";

/** How often an open window reports that it is still there. */
const HEARTBEAT = TimeSpan.minutes(5);

/**
 * Starts analytics for this window and returns the sink the app reports through. The
 * Console build returns the sink that discards everything, and the dead branch takes
 * every vendor with it.
 */
export const start = (): Analytics.Sink => {
  if (!DESKTOP || Session.Runtime.ENGINE !== "tauri") return Analytics.NOOP;
  const install = retrieveInstall();
  const sink = Analytics.createSink(create(identify(install)));
  // Every window reports its own screens and interactions, but a launch opens the app
  // once.
  if (Session.Runtime.isMainWindow()) void open(sink, install);
  beat(sink);
  return sink;
};

/** Names the install that posthog reports as. */
const identify = async (install: Promise<Install>): Promise<Params> => ({
  installID: (await install).id,
});

/** Reports the launch once the install and the version of the app are known. */
const open = async (sink: Analytics.Sink, install: Promise<Install>): Promise<void> => {
  try {
    const [record, version] = await Promise.all([install, getVersion()]);
    sink.capture("app_opened", {
      version,
      os: record.os,
      arch: record.arch,
      first_launch: record.firstLaunch,
      hours_since_last_launch: record.hoursSinceLastLaunch,
    });
  } catch (err) {
    console.error("failed to report the launch", err);
  }
};

/**
 * Reports that the window is open every {@link HEARTBEAT}, and whether anyone touched
 * it. A window left on a live plot earns no other event, so without this a day of
 * watching a test stand reads as nobody using Synnax.
 */
const beat = (sink: Analytics.Sink): void => {
  let interacted = false;
  const mark = (): void => {
    interacted = true;
  };
  const options = { capture: true, passive: true } as const;
  window.addEventListener("pointerdown", mark, options);
  window.addEventListener("keydown", mark, options);
  setInterval(() => {
    if (document.visibilityState !== "visible") return;
    sink.capture("app_active", { interacted });
    interacted = false;
  }, HEARTBEAT.milliseconds);
};
