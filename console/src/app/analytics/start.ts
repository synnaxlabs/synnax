// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { getVersion } from "@tauri-apps/api/app";

import { follow } from "@/app/analytics/core";
import { type Install, retrieveInstall } from "@/app/analytics/install";
import { create, type Params } from "@/app/analytics/posthog";
import { Analytics } from "@/platform/analytics";
import { Session } from "@/session";

/**
 * Starts analytics for this window and returns the sink the app reports through. The
 * Console build returns the sink that discards everything, and the dead branch takes
 * every vendor with it.
 */
export const start = (): Analytics.Sink => {
  if (!DESKTOP || Session.Runtime.ENGINE !== "tauri") return Analytics.NOOP;
  const install = retrieveInstall();
  const sink = Analytics.createSink(create(identify(install)));
  // Every window reports its own screens and actions, but a launch opens the app once,
  // and runs one embedded Core.
  if (Session.Runtime.isMainWindow()) {
    void open(sink, install);
    void follow(sink);
  }
  return sink;
};

/** Names the install that posthog reports as. */
const identify = async (install: Promise<Install>): Promise<Params> => ({
  installID: (await install).id,
});

/**
 * Reports the launch, and a reset that ran before it, once the install and the version
 * of the app are known.
 */
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
    // A reset restarts the app before posthog can send, so the launch after it reports.
    if (record.erasedBytes != null)
      sink.capture("core_reset", { data_size_bytes: record.erasedBytes });
  } catch (err) {
    console.error("failed to report the launch", err);
  }
};
