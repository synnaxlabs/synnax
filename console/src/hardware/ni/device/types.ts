// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";

import { type Common } from "@/hardware/common";

export const MAKE = "NI";
export type Make = typeof MAKE;

export type Properties = {
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
  counterInput: {
    portCount: number;
    index: channel.Key;
    channels: Record<string, channel.Key>;
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
  identifier: "",
  analogInput: { portCount: 0, index: 0, channels: {} },
  analogOutput: { portCount: 0, stateIndex: 0, channels: {} },
  counterInput: { portCount: 0, index: 0, channels: {} },
  digitalInputOutput: { portCount: 0, lineCounts: [] },
  digitalInput: { portCount: 0, lineCounts: [], index: 0, channels: {} },
  digitalOutput: { portCount: 0, lineCounts: [], stateIndex: 0, channels: {} },
};

export interface Device extends device.Device<Properties, Make> {}
export interface New extends device.New<Properties, Make> {}
