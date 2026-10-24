// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createIdentifier, createSlaveDevice } from "@/feature/ethercat/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const FILTER: EtherCAT.Device.StartupParameter = {
  index: 0x8000,
  subIndex: 0x15,
  dataType: "uint8",
  value: 3,
  name: "Filter",
};

const openEditor = async () => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  const slave = await createSlaveDevice(client, rack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    startupParameters: [FILTER],
  });
  await renderModalOpener(
    EtherCAT.Device.useStartupParametersModal,
    [{ deviceKey: slave.key }],
    { client },
  );
  await screen.findByLabelText("Index 1");
  return slave;
};

const setCell = (label: string, value: string) => {
  const input = screen.getByLabelText(label);
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const retrieveParameters = async (key: string) =>
  (await client.devices.retrieve({ key, schemas: EtherCAT.Device.SLAVE_SCHEMAS }))
    .properties.startupParameters;

describe("EtherCAT startup parameters modal", () => {
  it("should save edited and added parameters to the slave", async () => {
    const slave = await openEditor();
    expect(screen.getByLabelText<HTMLInputElement>("Index 1").value).toBe("0x8000");
    setCell("Value 1", "5");
    fireEvent.click(screen.getByRole("button", { name: "Add row" }));
    setCell("Index 2", "1C12");
    setCell("Subindex 2", "0");
    setCell("Value 2", "2");
    fireEvent.click(findButton("Save"));
    await waitFor(async () =>
      expect(await retrieveParameters(slave.key)).toEqual([
        { ...FILTER, value: 5 },
        { index: 0x1c12, subIndex: 0, dataType: "uint8", value: 2, name: "" },
      ]),
    );
  });

  it("should reject an index that is not hexadecimal without saving", async () => {
    const slave = await openEditor();
    setCell("Index 1", "zz");
    setCell("Value 1", "5");
    fireEvent.click(findButton("Save"));
    expect(
      await screen.findByText(
        "Row 1: the index must be a hexadecimal number such as 0x8000.",
      ),
    ).toBeTruthy();
    await act(async () => {});
    expect(await retrieveParameters(slave.key)).toEqual([FILTER]);
  });
});
