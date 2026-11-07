// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { strings } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createIdentifier, createSlaveDevice } from "@/feature/ethercat/testutil";
import { type Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Device } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const onCreateTask = vi.fn();
const onBack = vi.fn();

const useConfigureWithActionsModal = Modals.create<Device.ConfigureModalParams>(
  ({ deviceKey, close }) => (
    <EtherCAT.Device.HARDWARE.Configure
      deviceKey={deviceKey}
      close={close}
      onCreateTask={onCreateTask}
      onBack={onBack}
    />
  ),
);

const IDENTIFIER_FORMAT_ERROR =
  "Identifier must start with a letter and contain only letters, numbers, and underscores";

const OP_MODE = { index: 0, assignActivate: 0x0300, sync0Cycle: 0, sync1Cycle: 0 };

const FILTER: EtherCAT.Device.StartupParameter = {
  index: 0x8000,
  subIndex: 0x15,
  dataType: "uint16",
  value: 3,
  name: "Filter",
};

interface SetupArgs {
  configured?: boolean;
  useOpen?: typeof Device.useConfigureModal;
  analytics?: Analytics.Sink;
  startupParameters?: EtherCAT.Device.StartupParameter[];
}

const setup = async ({
  configured = true,
  useOpen = Device.useConfigureModal,
  analytics,
  startupParameters = [],
}: SetupArgs = {}) => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  const slave = await createSlaveDevice(
    client,
    rack.key,
    {
      name: uniqueName("slave"),
      identifier: configured ? createIdentifier() : "",
      opModes: [
        {
          name: "DC",
          assignActivate: 0x0300,
          sync0Cycle: 0,
          sync0Factor: 1,
          sync0Shift: 0,
          sync1Factor: 0,
          sync1Shift: 0,
        },
      ],
      opMode: OP_MODE,
      startupParameters,
    },
    configured,
  );
  await renderModalOpener(
    useOpen,
    [{ deviceKey: slave.key, hardware: EtherCAT.Device.HARDWARE }],
    { client, analytics },
  );
  await waitFor(() => expect(nameInput().value).toEqual(slave.name));
  return slave;
};

const nameInput = (): HTMLInputElement =>
  screen.getByRole<HTMLInputElement>("textbox", { name: "Name" });
const identifierInput = (): HTMLInputElement =>
  screen.getByRole<HTMLInputElement>("textbox", { name: /^Identifier/ });

const cell = (name: string): HTMLInputElement =>
  screen.getByRole<HTMLInputElement>("textbox", { name });

const commit = (name: string, value: string) => {
  const input = cell(name);
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const paste = (name: string, text: string) => {
  const input = cell(name);
  fireEvent.focus(input);
  fireEvent.paste(input, { clipboardData: { getData: () => text } });
};

const save = async (label = "Save") => {
  await act(async () => {
    fireEvent.click(findButton(label));
  });
};

const retrieve = async (key: string) =>
  await client.devices.retrieve({ key, schemas: EtherCAT.Device.SLAVE_SCHEMAS });

describe("EtherCAT.Device.HARDWARE.Configure", () => {
  beforeEach(() => {
    onCreateTask.mockClear();
    onBack.mockClear();
  });

  it("should save the name and identifier and keep the sync mode", async () => {
    const slave = await setup();
    const name = uniqueName("renamed");
    fireEvent.change(nameInput(), { target: { value: name } });
    fireEvent.change(identifierInput(), { target: { value: "slave_2" } });
    await save();
    await waitFor(async () => {
      const updated = await retrieve(slave.key);
      expect(updated.name).toEqual(name);
      expect(updated.properties.identifier).toEqual("slave_2");
      expect(updated.properties.opMode).toEqual(OP_MODE);
    });
    await waitFor(() => expect(screen.queryByText("General")).toBeNull());
  });

  it("should mark a new slave configured", async () => {
    const slave = await setup({ configured: false });
    fireEvent.change(identifierInput(), { target: { value: "slave_1" } });
    await save();
    await waitFor(async () =>
      expect((await retrieve(slave.key)).configured).toBe(true),
    );
  });

  it("should not save an identifier that does not start with a letter", async () => {
    const slave = await setup();
    fireEvent.change(identifierInput(), { target: { value: "1x" } });
    await save();
    await screen.findByText(IDENTIFIER_FORMAT_ERROR);
    const unchanged = await retrieve(slave.key);
    expect(unchanged.properties.identifier).toEqual(slave.properties.identifier);
  });

  it("should ask for a name and not save without one", async () => {
    const slave = await setup();
    fireEvent.change(nameInput(), { target: { value: "" } });
    await screen.findByText("Name is required");
    await save();
    expect(screen.getByText("Name is required")).toBeTruthy();
    expect((await retrieve(slave.key)).name).toEqual(slave.name);
  });

  it("should flag the empty identifier of a new slave only after Save", async () => {
    await setup({ configured: false });
    fireEvent.change(nameInput(), { target: { value: uniqueName("renamed") } });
    expect(screen.queryByText(IDENTIFIER_FORMAT_ERROR)).toBeNull();
    await save();
    await screen.findByText(IDENTIFIER_FORMAT_ERROR);
  });

  it("should fill the identifier from a suggestion", async () => {
    const slave = await setup();
    const [suggestion] = strings.createShortIdentifiers(slave.name);
    fireEvent.click(findButton(suggestion));
    expect(identifierInput().value).toEqual(suggestion);
  });

  it("should report the make of a slave it configures for the first time", async () => {
    const analytics = createTestSink();
    await setup({ configured: false, analytics });
    fireEvent.change(identifierInput(), { target: { value: "slave_1" } });
    await save();
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_configured", {
        make: EtherCAT.Device.MAKE,
      }),
    );
  });

  it("should not report a change to a configured slave", async () => {
    const analytics = createTestSink();
    const slave = await setup({ analytics });
    const name = uniqueName("renamed");
    fireEvent.change(nameInput(), { target: { value: name } });
    await save();
    await waitFor(async () => expect((await retrieve(slave.key)).name).toEqual(name));
    await waitFor(() => expect(screen.queryByText("General")).toBeNull());
    expect(analytics.capture).not.toHaveBeenCalledWith(
      "device_configured",
      expect.anything(),
    );
  });

  describe("startup parameters", () => {
    it("should show the address of a saved parameter in hexadecimal", async () => {
      await setup({ startupParameters: [FILTER] });
      expect(cell("Index 1").value).toEqual("0x8000");
      expect(cell("Subindex 1").value).toEqual("0x15");
      expect(cell("Name 1").value).toEqual("Filter");
    });

    it("should save a parameter added in the table", async () => {
      const slave = await setup();
      fireEvent.click(findButton("Add parameter"));
      commit("Index 1", "8000");
      commit("Subindex 1", "0x15");
      commit("Value 1", "3");
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          { index: 0x8000, subIndex: 0x15, dataType: "uint8", value: 3, name: "" },
        ]),
      );
    });

    it("should keep the last address when the text is not hexadecimal", async () => {
      await setup({ startupParameters: [FILTER] });
      commit("Index 1", "0x8g00");
      commit("Subindex 1", "0x100");
      expect(cell("Index 1").value).toEqual("0x8000");
      expect(cell("Subindex 1").value).toEqual("0x15");
    });

    it("should read a pasted address as hexadecimal", async () => {
      const slave = await setup();
      fireEvent.click(findButton("Add parameter"));
      paste("Index 1", "8000\t15\tuint16\t3\tFilter");
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          FILTER,
        ]),
      );
    });

    it("should read a TwinCAT data type name", async () => {
      const slave = await setup();
      fireEvent.click(findButton("Add parameter"));
      paste("Index 1", "8000\t15\tUINT\t3\tFilter");
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          FILTER,
        ]),
      );
    });

    it("should give a parameter that a paste adds the default data type", async () => {
      const slave = await setup();
      fireEvent.click(findButton("Add parameter"));
      paste("Index 1", "8000\t15\n8001\t16");
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          { index: 0x8000, subIndex: 0x15, dataType: "uint8", value: 0, name: "" },
          { index: 0x8001, subIndex: 0x16, dataType: "uint8", value: 0, name: "" },
        ]),
      );
    });

    it("should keep the data type when the pasted one is unknown", async () => {
      const slave = await setup();
      fireEvent.click(findButton("Add parameter"));
      paste("Index 1", "8000\t15\tSTRING\t3");
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          { index: 0x8000, subIndex: 0x15, dataType: "uint8", value: 3, name: "" },
        ]),
      );
    });

    it("should not save a value that the data type cannot hold", async () => {
      const slave = await setup({ startupParameters: [FILTER] });
      commit("Value 1", "70000");
      await save();
      await waitFor(() =>
        expect(cell("Value 1").getAttribute("aria-invalid")).toEqual("true"),
      );
      expect(screen.getByText("Startup parameters")).toBeTruthy();
      expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
        FILTER,
      ]);
    });

    it("should remove a parameter", async () => {
      const slave = await setup({ startupParameters: [FILTER] });
      fireEvent.click(screen.getByRole("button", { name: "Remove parameter 1" }));
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([]),
      );
    });
  });

  describe("actions", () => {
    it("should create a read task for a new slave only after it saves", async () => {
      const slave = await setup({
        configured: false,
        useOpen: useConfigureWithActionsModal,
      });
      fireEvent.change(identifierInput(), { target: { value: "slave_1" } });
      await save("Save and create read task");
      await waitFor(() => expect(onCreateTask).toHaveBeenCalledWith(slave.key));
      expect((await retrieve(slave.key)).configured).toBe(true);
    });

    it("should not offer a read task for a configured slave", async () => {
      await setup({ useOpen: useConfigureWithActionsModal });
      expect(findButton("Back")).toBeTruthy();
      expect(screen.queryByText("Save and create read task")).toBeNull();
    });

    it("should go back without saving", async () => {
      const slave = await setup({
        configured: false,
        useOpen: useConfigureWithActionsModal,
      });
      fireEvent.change(identifierInput(), { target: { value: "slave_1" } });
      fireEvent.click(findButton("Back"));
      expect(onBack).toHaveBeenCalledOnce();
      expect((await retrieve(slave.key)).configured).toBe(false);
    });
  });
});
