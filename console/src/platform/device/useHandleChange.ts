// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type Form } from "@synnaxlabs/lyra/form";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { deep, primitive } from "@synnaxlabs/x";

const path = deep.path<{ rack: number }>();

/**
 * Returns what a task form runs after its device changes: it sets the task's rack to
 * the device's rack, and calls onConfigure when the device is not configured.
 */
export const useHandleChange = (
  onConfigure: (key: device.Key) => void,
): ((key: device.Key, ctx: Form.ContextValue) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return (key, { set }) => {
    if (client == null || primitive.isZero(key)) return;
    handleError(async () => {
      const { configured, rack } = await client.devices.retrieve({ key });
      set(path.rack, rack);
      if (!configured) onConfigure(key);
    }, "Failed to retrieve device");
  };
};
