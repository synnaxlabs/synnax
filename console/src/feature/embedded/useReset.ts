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

import { reset } from "@/feature/embedded/supervisor";
import { Modals } from "@/platform/modals";

const HOLD_TO_CONFIRM_MS = 1000;

/**
 * Returns a callback that erases everything Synnax Desktop has stored and starts the
 * app again. It asks first, and the person must hold the button to confirm.
 */
export const useReset = (): (() => void) => {
  const confirm = Modals.useConfirm();
  const handleError = Status.useErrorHandler();
  return useCallback(
    () =>
      handleError(async () => {
        const confirmed = await confirm({
          message: "Are you sure you want to erase all data?",
          description:
            "This permanently deletes every channel, all recorded data, and every project, and then restarts Synnax. It cannot be undone. Hold the button to confirm.",
          confirm: { label: "Erase all data", delay: HOLD_TO_CONFIRM_MS },
        });
        if (confirmed !== true) return;
        await reset();
      }, "Failed to erase the data"),
    [confirm, handleError],
  );
};
