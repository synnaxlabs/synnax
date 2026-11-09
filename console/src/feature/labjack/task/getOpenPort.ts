// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type AIPort,
  type AIPortType,
  type AOPort,
  type AOPortType,
  type DIPort,
  type DIPortType,
  type DOPort,
  type DOPortType,
  type Model,
  PORTS,
  type PortType,
} from "@/feature/labjack/device/types";
import { claims, terminal } from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";

// This is a bit of a confusing type, but basically it maps the port type (AI, AO, DI,
// DO) to the actual port object (AIPort, AOPort, DIPort, DOPort).
export type Port<T extends PortType> = T extends AIPortType
  ? AIPort
  : T extends DIPortType
    ? DIPort
    : T extends DOPortType
      ? DOPort
      : T extends AOPortType
        ? AOPort
        : never;

export const getOpenPort = <T extends PortType>(
  channels: Channel[],
  model: Model,
  types: T[],
): Port<T> | null => {
  const taken = claims(channels, model);
  for (const type of types) {
    const port = PORTS[model][type].find((p) => !taken.has(terminal(p)));
    if (port != null) return port as Port<T>;
  }
  return null;
};
