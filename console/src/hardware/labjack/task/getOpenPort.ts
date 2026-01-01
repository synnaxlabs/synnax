// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/hardware/labjack/device";
import { type Channel } from "@/hardware/labjack/task/types";

// This is a bit of a confusing type, but basically it maps the port type (AI, AO, DI,
// DO) to the actual port object (AIPort, AOPort, DIPort, DOPort).
export type Port<T extends Device.PortType> = T extends Device.AIPortType
  ? Device.AIPort
  : T extends Device.DIPortType
    ? Device.DIPort
    : T extends Device.DOPortType
      ? Device.DOPort
      : T extends Device.AOPortType
        ? Device.AOPort
        : never;

export const getOpenPort = <T extends Device.PortType>(
  channels: Channel[],
  model: Device.Model,
  types: T[],
): Port<T> | null => {
  const portsInUse = new Set(channels.map(({ port }) => port));
  for (const type of types) {
    const port = Device.PORTS[model][type].find(({ key }) => !portsInUse.has(key));
    if (port != null) return port as Port<T>;
  }
  return null;
};
