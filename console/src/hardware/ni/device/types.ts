// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";

import { type Common } from "@/hardware/common";

export const MAKE = "ni";

export type PropertiesDigest = { key: string; enriched: boolean };

export type Properties = PropertiesDigest & {
  identifier: Common.Device.Identifier;
  analogInput: {
    portCount: number;
    index: channel.Key;
    channels: Record<string, channel.Key>;
  };
  analogOutput: {
    portCount: number;
    stateIndex: channel.Key;
    channels: Record<string, Common.Device.CommandStatePair>;
  };
  digitalInputOutput: { portCount: number; lineCounts: number[] };
  digitalInput: {
    portCount: number;
    lineCounts: number[];
    index: channel.Key;
    channels: Record<string, channel.Key>;
  };
  digitalOutput: {
    portCount: number;
    lineCounts: number[];
    stateIndex: channel.Key;
    channels: Record<string, Common.Device.CommandStatePair>;
  };
};

export const ZERO_PROPERTIES: Properties = {
  key: "",
  enriched: false,
  identifier: "",
  analogInput: { portCount: 0, index: 0, channels: {} },
  analogOutput: { portCount: 0, stateIndex: 0, channels: {} },
  digitalInputOutput: { portCount: 0, lineCounts: [] },
  digitalInput: { portCount: 0, lineCounts: [], index: 0, channels: {} },
  digitalOutput: { portCount: 0, lineCounts: [], stateIndex: 0, channels: {} },
};

export interface Device extends device.Device<Properties> {}
