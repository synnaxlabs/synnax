// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Haul, type Status } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import {
  CaptureStatuses,
  createConsoleWrapper,
  isSelectButtonSelected,
} from "@/testutil";

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

const TAG: MQTT.Device.SparkplugHaulTag = {
  group: "plant",
  edgeNode: "line1",
  device: "ovenA",
  tag: "zone 1/temperature",
  dataType: "double",
};

describe("mqtt device Sparkplug B haul utilities", () => {
  describe("createSparkplugHaulItem", () => {
    it("builds an item keyed by the identity of the tag", () => {
      expect(MQTT.Device.createSparkplugHaulItem(TAG)).toEqual({
        type: "mqtt-sparkplug-tag",
        key: "plant/line1/ovenA/zone 1/temperature",
        data: TAG,
      });
    });

    it("keys a tag of the edge node apart from a tag of a device", () => {
      const nodeTag = { ...TAG, device: "", tag: "ovenA/zone 1/temperature" };
      expect(MQTT.Device.createSparkplugHaulItem(nodeTag).key).toBe(
        "plant/line1//ovenA/zone 1/temperature",
      );
    });

    it("drops the fields of a browsed tag that a task does not take", () => {
      const browsed = { ...TAG, value: "21.5", supported: true };
      expect(MQTT.Device.createSparkplugHaulItem(browsed).data).toEqual(TAG);
    });
  });

  describe("isSparkplugHaulItem", () => {
    it("tells a tag item from a topic item", () => {
      const item = MQTT.Device.createSparkplugHaulItem(TAG);
      expect(MQTT.Device.isSparkplugHaulItem(item)).toBe(true);
      expect(MQTT.Device.isSparkplugHaulItem(MQTT.Device.createHaulItem(TOPIC))).toBe(
        false,
      );
      expect(MQTT.Device.isHaulItem(item)).toBe(false);
    });
  });

  describe("filterSparkplugHaulItems", () => {
    it("keeps tag items and drops items of other kinds", () => {
      const item = MQTT.Device.createSparkplugHaulItem(TAG);
      const topic = MQTT.Device.createHaulItem(TOPIC);
      expect(MQTT.Device.filterSparkplugHaulItems([item, topic, OTHER])).toEqual([
        item,
      ]);
    });
  });

  describe("canDropSparkplugHaulItem", () => {
    it("returns true when at least one item is a tag item", () => {
      expect(
        MQTT.Device.canDropSparkplugHaulItem({
          source: OTHER,
          items: [MQTT.Device.createSparkplugHaulItem(TAG), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is a tag item", () => {
      expect(
        MQTT.Device.canDropSparkplugHaulItem({
          source: OTHER,
          items: [MQTT.Device.createHaulItem(TOPIC)],
        }),
      ).toBe(false);
    });
  });
});

const renderBrowser = async (client: Synnax = createTestClient()) => {
  const dev = await client.devices.retrieve({
    key: (await createBroker(client)).key,
    schemas: MQTT.Device.SCHEMAS,
  });
  const { wrapper } = await createConsoleWrapper({ client });
  const captured: { statuses: Status.NotificationSpec[] } = { statuses: [] };
  render(
    <>
      <MQTT.Device.Browser device={dev} />
      <CaptureStatuses onStatuses={(s) => (captured.statuses = s)} />
    </>,
    { wrapper },
  );
  return { captured, dev };
};

const NODES = [{ group: "plant", edgeNode: "line_1", devices: ["ovenA"] }];

const TAGS = [
  { device: "", name: "Node Control/Rebirth", dataType: "boolean", value: "false" },
  { device: "ovenA", name: "zoneTemp", dataType: "double", value: "21.5" },
  { device: "ovenA", name: "recipe", dataType: "DataSet", value: "" },
].map((t) => ({ ...t, supported: t.dataType !== "DataSet" }));

describe("Browser", () => {
  it("should surface a scan task retrieval failure as an error status", async () => {
    const { captured } = await renderBrowser();
    const browse = await screen.findByRole("button", { name: "Browse" });
    fireEvent.click(browse);
    await screen.findByText(/not found/i);
    expect(captured.statuses).toHaveLength(0);
    fireEvent.click(browse);
    await screen.findByText(/not found/i);
    expect(captured.statuses).toHaveLength(0);
  });

  it("should open on the topics and switch to the Sparkplug B edge nodes", async () => {
    await renderBrowser();
    expect(await screen.findByPlaceholderText("#")).toBeTruthy();
    expect(isSelectButtonSelected("Topics")).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Sparkplug B" }));
    expect(await screen.findByPlaceholderText("All groups")).toBeTruthy();
    expect(screen.queryByPlaceholderText("#")).toBeNull();
    expect(screen.getByText(/No edge nodes/)).toBeTruthy();
    expect(screen.getAllByRole("button", { name: "Browse" })).toHaveLength(1);
  });

  describe("with a scan task that answers", () => {
    const browseNodes = async () => {
      const client = createTestClient();
      const { dev } = await renderBrowser(client);
      const scanner = await openScanner(client, dev.rack);
      fireEvent.click(await screen.findByRole("button", { name: "Sparkplug B" }));
      const group = await screen.findByPlaceholderText("All groups");
      fireEvent.change(group, { target: { value: "plant" } });
      fireEvent.click(screen.getByRole("button", { name: "Browse" }));
      const cmd = await scanner.answer({ data: { nodes: NODES, tags: [] } });
      return { dev, cmd, scanner };
    };

    it("should list the edge nodes that the group filter selects", async () => {
      const { dev, cmd, scanner } = await browseNodes();
      scanner.close();
      expect(cmd.type).toBe("browse_sparkplug");
      // The streamer reads the snake case keys of the wire as camel case.
      expect(cmd.args).toMatchObject({ device: dev.key, group: "plant", edgeNode: "" });
      expect(await screen.findByText("plant/line_1")).toBeTruthy();
    });

    it("should list the tags of an edge node under their devices", async () => {
      const { scanner } = await browseNodes();
      fireEvent.click(await screen.findByText("plant/line_1"));
      const cmd = await scanner.answer({ data: { nodes: [], tags: TAGS } });
      scanner.close();
      expect(cmd.args).toMatchObject({ group: "plant", edgeNode: "line_1" });
      const rebirth = await screen.findByText("Node Control/Rebirth");
      expect(screen.getByText("false · boolean")).toBeTruthy();
      expect(rebirth.closest("[draggable='true']")).not.toBeNull();
      expect(screen.queryByText("zoneTemp")).toBeNull();
      fireEvent.click(screen.getByText("ovenA"));
      expect(await screen.findByText("zoneTemp")).toBeTruthy();
      expect(screen.getByText("21.5 · double")).toBeTruthy();
    });

    it("should disable a tag that a task cannot take", async () => {
      const { scanner } = await browseNodes();
      fireEvent.click(await screen.findByText("plant/line_1"));
      await scanner.answer({ data: { nodes: [], tags: TAGS } });
      scanner.close();
      fireEvent.click(await screen.findByText("ovenA"));
      const item = (await screen.findByText("recipe")).closest("[role='treeitem']");
      expect(item?.getAttribute("aria-disabled")).toBe("true");
      expect(item?.getAttribute("draggable")).not.toBe("true");
      expect(screen.getByText("DataSet")).toBeTruthy();
    });

    it("should keep the edge nodes when one does not answer", async () => {
      const { scanner } = await browseNodes();
      fireEvent.click(await screen.findByText("plant/line_1"));
      const message = "edge node plant/line_1 did not answer the rebirth request";
      await scanner.answer({ variant: "error", message });
      scanner.close();
      expect(await screen.findByText(message)).toBeTruthy();
      expect(screen.getByText("plant/line_1")).toBeTruthy();
    });
  });

  it("should surface a failed browse for edge nodes as an error status", async () => {
    const { captured } = await renderBrowser();
    fireEvent.click(await screen.findByRole("button", { name: "Sparkplug B" }));
    fireEvent.click(await screen.findByRole("button", { name: "Browse" }));
    await screen.findByText(/not found/i);
    expect(captured.statuses).toHaveLength(0);
  });
});
