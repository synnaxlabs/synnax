// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useStatus } from "@/feature/embedded/Provider";
import { restart } from "@/feature/embedded/supervisor";
import { Modals } from "@/platform/modals";

/**
 * Returns a callback that restarts the embedded Core. It asks first while the Core
 * runs, because a restart interrupts every task and every live plot.
 */
export const useRestart = (): (() => void) => {
  const { state } = useStatus();
  const confirm = Modals.useConfirm();
  const handleError = Status.useErrorHandler();
  return useCallback(
    () =>
      handleError(async () => {
        if (state === "running") {
          const confirmed = await confirm({
            message: "Are you sure you want to restart Synnax?",
            description:
              "Running tasks and live data stop for a few seconds. No saved data is lost.",
            confirm: { label: "Restart", variant: "warning" },
          });
          if (confirmed !== true) return;
        }
        await restart();
      }, "Failed to restart Synnax"),
    [state, confirm, handleError],
  );
};
