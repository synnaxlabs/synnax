// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type rack } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/ni/task/types";

/** Toggles the enabled state of the NI device scanner on a specific rack. */
export const useToggleScanner = (rackKey: rack.Key) => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  return useCallback(
    () =>
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const { config, payload } = await client.tasks.retrieve({
          type: SCAN_TYPE,
          rack: rackKey,
          schemas: SCAN_SCHEMAS,
        });
        const {
          config: { enabled },
        } = await client.tasks.create(
          { ...payload, config: { ...config, enabled: !config.enabled } },
          SCAN_SCHEMAS,
        );
        addStatus({
          variant: "success",
          message: `NI device scanning ${enabled ? "enabled" : "disabled"}`,
        });
      }, "Failed to toggle NI device scanner"),
    [client, addStatus, handleError, rackKey],
  );
};
