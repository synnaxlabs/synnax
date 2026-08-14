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

import { OPC } from "@/feature/opc";
import { createOPCDevice } from "@/feature/opc/testutil";
import { CaptureStatuses, createConsoleWrapper, getIconButton } from "@/testutil";

const NODE: OPC.Task.ScannedNode = {
  key: "ns=2;s=Demo.Static.Scalar.Float",
  nodeId: "ns=2;s=Demo.Static.Scalar.Float",
  name: "Demo Float",
  nodeClass: "Variable",
  dataType: "float32",
  isArray: false,
};

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("opc device haul utilities", () => {
  describe("createHaulItem", () => {
    it("builds an item keyed by node ID carrying the full scanned node", () => {
      expect(OPC.Device.createHaulItem(NODE)).toEqual({
        type: OPC.Device.HAUL_TYPE,
        key: NODE.nodeId,
        data: NODE,
      });
    });
  });

  describe("filterHaulItems", () => {
    it("keeps opc items and drops items of other kinds", () => {
      const item = OPC.Device.createHaulItem(NODE);
      expect(OPC.Device.filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is an opc item", () => {
      expect(
        OPC.Device.canDropHaulItem({
          source: OTHER,
          items: [OPC.Device.createHaulItem(NODE), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is an opc item", () => {
      expect(OPC.Device.canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});

describe("Browser", () => {
  it("should surface a scan task retrieval failure as an error status", async () => {
    const client = createTestClient();
    const dev = await createOPCDevice(client);
    const { wrapper } = await createConsoleWrapper({ client });
    let statuses: Status.NotificationSpec[] = [];
    const { container } = render(
      <>
        <OPC.Device.Browser device={dev} />
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </>,
      { wrapper },
    );
    await screen.findByText("Browser");
    await screen.findByText(/not found/i);
    expect(statuses).toHaveLength(0);
    const refresh = getIconButton(container, "refresh");
    expect(refresh.disabled).toBe(false);
    fireEvent.click(refresh);
    await screen.findByText(/not found/i);
    expect(statuses).toHaveLength(0);
  });
});
