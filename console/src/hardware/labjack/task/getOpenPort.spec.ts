// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Device } from "@/hardware/labjack/device";
import { getOpenPort } from "@/hardware/labjack/task/getOpenPort";
import {
  AI_CHANNEL_TYPE,
  AO_CHANNEL_TYPE,
  type AOChannelType,
  type Channel,
  DI_CHANNEL_TYPE,
  ZERO_INPUT_CHANNELS,
  ZERO_OUTPUT_CHANNEL,
} from "@/hardware/labjack/task/types";

describe("getOpenPort", () => {
  // We'll use the T4 model for our tests.
  const model: Device.Model = Device.T4_MODEL;

  it("returns the first available port for a given type when none are in use", () => {
    const channels: Channel[] = [];
    const type = Device.AI_PORT_TYPE;
    const port = getOpenPort(channels, model, [type]);

    // The expected port is the first port in the list for the given type.
    const expectedPort = Device.PORTS[model][type][0];
    expect(port).toEqual(expectedPort);
  });

  it("skips ports that are already in use", () => {
    const type = Device.AI_PORT_TYPE;
    const aiPorts = Device.PORTS[model][type];
    // Mark the first port as in use.
    const channels: Channel[] = [
      { ...ZERO_INPUT_CHANNELS[AI_CHANNEL_TYPE], port: aiPorts[0].key },
    ];
    const port = getOpenPort(channels, model, [type]);

    // The expected port is the second port from the AI ports list.
    const expectedPort = aiPorts[1];
    expect(port).toEqual(expectedPort);
  });

  it("returns null if all ports for the given type are in use", () => {
    const type = Device.AI_PORT_TYPE;
    const aiPorts = Device.PORTS[model][type];
    // Mark every port for this type as in use.
    const channels: Channel[] = aiPorts.map(({ key }) => ({
      ...ZERO_INPUT_CHANNELS[AI_CHANNEL_TYPE],
      port: key,
    }));
    const port = getOpenPort(channels, model, [type]);
    expect(port).toBeNull();
  });

  it("returns the first available port from the first type that has an available port when multiple types are provided", () => {
    // For this test, we supply two port types.
    // Mark all DI ports as in use and leave AO ports free.
    const type1 = Device.DI_PORT_TYPE;
    const type2 = Device.AO_PORT_TYPE;
    const diPorts = Device.PORTS[model][type1];
    const channels: Channel[] = diPorts.map(({ key }) => ({
      ...ZERO_INPUT_CHANNELS[DI_CHANNEL_TYPE],
      port: key,
    }));

    const port = getOpenPort(channels, model, [type1, type2]);
    // Since all DI ports are taken, we expect the first AO port.
    const expectedPort = Device.PORTS[model][type2][0];
    expect(port).toEqual(expectedPort);
  });

  it("returns null when multiple types are provided but all ports are in use", () => {
    const type1 = Device.AI_PORT_TYPE;
    const type2 = Device.AO_PORT_TYPE;
    const aiPorts = Device.PORTS[model][type1];
    const aoPorts = Device.PORTS[model][type2];

    // Mark all ports for both AI and AO as in use.
    const channels: Channel[] = [
      ...aiPorts.map(({ key }) => ({
        ...ZERO_INPUT_CHANNELS[AI_CHANNEL_TYPE],
        port: key,
      })),
      ...aoPorts.map(({ key }) => ({
        ...ZERO_OUTPUT_CHANNEL,
        type: AO_CHANNEL_TYPE as AOChannelType,
        port: key,
      })),
    ];

    const port = getOpenPort(channels, model, [type1, type2]);
    expect(port).toBeNull();
  });
});
