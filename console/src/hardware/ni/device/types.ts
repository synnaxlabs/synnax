// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";

export type PropertiesDigest = { key: string; enriched: boolean };

interface CommandStatePair {
  command: channel.Key;
  state: channel.Key;
}

export type Properties = PropertiesDigest & {
  identifier: string;
  analogInput: {
    portCount: number;
    index: number;
    channels: Record<string, channel.Key>;
  };
  analogOutput: { portCount: number };
  digitalInputOutput: { portCount: number; lineCounts: number[] };
  digitalInput: {
    portCount: number;
    lineCounts: number[];
    index: number;
    channels: Record<string, channel.Key>;
  };
  digitalOutput: {
    portCount: number;
    lineCounts: number[];
    stateIndex: number;
    channels: Record<string, CommandStatePair>;
  };
};

export const ZERO_PROPERTIES: Properties = {
  key: "",
  enriched: false,
  identifier: "",
  analogInput: { portCount: 0, index: 0, channels: {} },
  analogOutput: { portCount: 0 },
  digitalInputOutput: { portCount: 0, lineCounts: [] },
  digitalInput: { portCount: 0, lineCounts: [], index: 0, channels: {} },
  digitalOutput: { portCount: 0, lineCounts: [], stateIndex: 0, channels: {} },
};

export interface Device extends device.Device<Properties> {}
