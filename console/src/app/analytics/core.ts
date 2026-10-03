// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Embedded } from "@/feature/embedded";
import { type Analytics } from "@/platform/analytics";

/**
 * Reports what the embedded Core does. The counts in the history decide what is new, so
 * the first read reports a Core that was already up before this window loaded, and a
 * second read of the same history reports nothing.
 */
export const follow = async (sink: Analytics.Sink): Promise<void> => {
  let readies = 0;
  let exits = 0;
  let failures = 0;
  const report = async (): Promise<void> => {
    try {
      const history = await Embedded.retrieveHistory();
      const { lastExit } = history;
      if (history.exits > exits) {
        exits = history.exits;
        if (lastExit != null)
          sink.capture("core_exited", {
            reason: lastExit.reason,
            uptime_seconds: lastExit.uptimeSeconds,
          });
      }
      if (history.readies > readies && history.timeToReadyMs != null) {
        readies = history.readies;
        sink.capture("core_ready", {
          time_to_ready_ms: history.timeToReadyMs,
          starts: history.starts,
        });
      }
      if (history.failures > failures) {
        failures = history.failures;
        if (lastExit != null)
          sink.capture("core_restart_exhausted", { reason: lastExit.reason });
      }
    } catch (err) {
      console.error("failed to report the state of the Core", err);
    }
  };
  try {
    await Embedded.onStatusChange(() => void report());
    // The Core can pass its probe before this window finishes loading, so what it did
    // before counts as much as the changes that follow.
    await report();
  } catch (err) {
    console.error("failed to follow the Core", err);
  }
};
