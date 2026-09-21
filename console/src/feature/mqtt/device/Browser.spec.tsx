// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Haul, type Status } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker } from "@/feature/mqtt/testutil";
import { CaptureStatuses, createConsoleWrapper } from "@/testutil";

const TOPIC: MQTT.Task.BrowsedTopic = {
  topic: "plant/line1/temperature",
  payload: '{"value": 21.5}',
  retained: true,
};

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("mqtt device haul utilities", () => {
  describe("createHaulItem", () => {
    it("builds an item keyed by topic carrying the browsed topic", () => {
      expect(MQTT.Device.createHaulItem(TOPIC)).toEqual({
        type: MQTT.Device.HAUL_TYPE,
        key: TOPIC.topic,
        data: TOPIC,
      });
    });
  });

  describe("isHaulItem", () => {
    it("tells an mqtt item from an item of another kind", () => {
      expect(MQTT.Device.isHaulItem(MQTT.Device.createHaulItem(TOPIC))).toBe(true);
      expect(MQTT.Device.isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps mqtt items and drops items of other kinds", () => {
      const item = MQTT.Device.createHaulItem(TOPIC);
      expect(MQTT.Device.filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is an mqtt item", () => {
      expect(
        MQTT.Device.canDropHaulItem({
          source: OTHER,
          items: [MQTT.Device.createHaulItem(TOPIC), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is an mqtt item", () => {
      expect(MQTT.Device.canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(
        false,
      );
    });
  });
});

describe("Browser", () => {
  it("should surface a scan task retrieval failure as an error status", async () => {
    const client = createTestClient();
    const dev = await client.devices.retrieve({
      key: (await createBroker(client)).key,
      schemas: MQTT.Device.SCHEMAS,
    });
    const { wrapper } = await createConsoleWrapper({ client });
    let statuses: Status.NotificationSpec[] = [];
    render(
      <>
        <MQTT.Device.Browser device={dev} />
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </>,
      { wrapper },
    );
    const browse = await screen.findByRole("button", { name: "Browse" });
    fireEvent.click(browse);
    await screen.findByText(/not found/i);
    expect(statuses).toHaveLength(0);
    fireEvent.click(browse);
    await screen.findByText(/not found/i);
    expect(statuses).toHaveLength(0);
  });
});
