// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/hardware/labjack/device";
import { type Channel } from "@/hardware/labjack/task/types";

export const getOpenPort = <T extends Device.PortType>(
  channels: Channel[],
  model: Device.Model,
  types: T[],
): T extends Device.AIPortType
  ? Device.AIPort | null
  : T extends Device.DIPortType
    ? Device.DIPort | null
    : T extends Device.DOPortType
      ? Device.DOPort | null
      : T extends Device.AOPortType
        ? Device.AOPort | null
        : null => {
  const portsInUse = new Set(channels.map(({ port }) => port));
  // @ts-expect-error TypeScript cannot properly infer the type of the return value.
  return (
    types.find(
      (type) =>
        Device.DEVICES[model].ports[type].find(({ key }) => !portsInUse.has(key)) !=
        null,
    ) ?? null
  );
};
