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
  model: Device.ModelKey,
  types: T[],
): T extends "AI"
  ? Device.AIPort | null
  : T extends "DI"
    ? Device.DIPort | null
    : T extends "DO"
      ? Device.DOPort | null
      : T extends "AO"
        ? Device.AOPort | null
        : null => {
  const portsInUse = new Set(channels.map((c) => c.port));
  for (const type of types) {
    const port = Device.DEVICES[model].ports[type].find(
      (port) => !portsInUse.has(port.key),
    );
    // This is safe because we narrow the type explicitly.
    if (port != null) return port as any;
  }
  // @ts-expect-error TypeScript cannot properly infer the type of the return value.
  return null;
};
