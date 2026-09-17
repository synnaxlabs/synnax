// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { primitive, unique } from "@synnaxlabs/x";
import { useMemo } from "react";

interface DeviceEntry {
  device: device.Key;
}

/** The distinct device keys the form's channel entries name, sorted. */
export const useChannelDeviceKeys = (path = "config.channels"): device.Key[] => {
  const state = Form.useFieldState<DeviceEntry[]>(path, { optional: true });
  return useMemo(
    () =>
      unique
        .unique((state?.value ?? []).map(({ device }) => device))
        .filter(primitive.isNonZero)
        .sort(),
    [state],
  );
};
