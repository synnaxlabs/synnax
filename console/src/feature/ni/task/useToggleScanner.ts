// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, NotFoundError, type rack, task } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";

import { SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/ni/task/types";

/** Toggles the enabled state of the NI device scanner on a specific rack. */
export const useToggleScanner = (rackKey: rack.Key) => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  return () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      // A retrieve with schemas skips the cache, and a dispatch needs the cached task.
      const scanner = await client.tasks.retrieve({
        type: SCAN_TYPE,
        rack: rackKey,
        one: true,
      });
      const disabled = !SCAN_SCHEMAS.config.parse(scanner.config).disabled;
      const set = task.set({ path: ["config", "disabled"], value: disabled });
      if (!(await client.tasks.dispatch(scanner.key, set)))
        throw new NotFoundError(`Task ${scanner.key} is not cached`);
      addStatus({
        variant: "success",
        message: `${disabled ? "Disabled" : "Enabled"} NI device scanning`,
      });
    }, "Failed to toggle NI device scanner");
};
