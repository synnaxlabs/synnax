// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { strings } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createDewesoftXSettings,
  createIdentifier,
  createSettingsDescription,
  createSlaveDevice,
  STRAIN_ENTRY,
  STRAIN_SAMPLE_ENTRY,
} from "@/feature/ethercat/testutil";
import { type Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Device } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { fakePickedFile, interceptFilePicker, uniqueName } from "@/testutil";

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
  settings?: EtherCAT.Device.SlaveProperties["settings"];
  inputs?: ethercat.PDOEntry[];
  productCode?: number;
  revision?: number;
}

const setup = async ({
  configured = true,
  useOpen = Device.useConfigureModal,
  analytics,
  startupParameters = [],
  settings,
  inputs = [],
  productCode = 0,
  revision = 0,
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
      settings,
      pdos: { inputs, outputs: [] },
      productCode,
      revision,
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

    describe("import from an ESI file", () => {
      const initCmd = (
        transition: string,
        index: string,
        subIndex: string,
        data: string,
        comment: string,
        attributes = "",
      ) => `<InitCmd${attributes}>
        <Transition>${transition}</Transition>
        <Index>${index}</Index>
        <SubIndex>${subIndex}</SubIndex>
        <Data>${data}</Data>
        <Comment>${comment}</Comment>
      </InitCmd>`;

      const ESI = `<?xml version="1.0" encoding="utf-8"?>
      <EtherCATInfo>
        <Descriptions>
          <Devices>
            <Device>
              <Type ProductCode="252" RevisionNo="#x00010000">IOLITEr 6xSTG</Type>
              <Mailbox>
                <CoE>
                  ${initCmd("IP", "32800", "9", "0100", "Bus1 mode")}
                  ${initCmd("IP", "32800", "10", "0300", "Bus2 mode")}
                </CoE>
              </Mailbox>
            </Device>
            <Device>
              <Type ProductCode="#x66" RevisionNo="#x01000001">SIRIUSe 8xAI</Type>
              <Mailbox>
                <CoE>
                  ${initCmd("IP", "#x8020", "07", "0600", "OperationMode")}
                  ${initCmd("PS", "#x8040", "#x06", "0000a040", "Excitation")}
                  ${initCmd("SO", "#x8040", "1", "01", "Starts in OP")}
                  ${initCmd("PS", "#x1c12", "0", "000000", "Three bytes")}
                  ${initCmd("PS", "#x1c13", "0", "0100", "Complete", ' CompleteAccess="1"')}
                </CoE>
              </Mailbox>
            </Device>
          </Devices>
        </Descriptions>
      </EtherCATInfo>`;

      const SIRIUS = { productCode: 0x66, revision: 0x01000001 };
      const OPERATION_MODE: EtherCAT.Device.StartupParameter = {
        index: 0x8020,
        subIndex: 7,
        dataType: "uint16",
        value: 6,
        name: "OperationMode",
      };
      const EXCITATION: EtherCAT.Device.StartupParameter = {
        index: 0x8040,
        subIndex: 6,
        dataType: "uint32",
        value: 0x40a00000,
        name: "Excitation",
      };

      const importFile = (name: string, contents: string) => {
        const picker = interceptFilePicker();
        fireEvent.click(findButton("Import from ESI file"));
        picker.selectFiles([fakePickedFile(name, contents)]);
      };

      afterEach(() => vi.restoreAllMocks());

      it("should add the commands that the device runs before SAFE-OP", async () => {
        const slave = await setup(SIRIUS);
        importFile("sirius.xml", ESI);
        expect(
          await screen.findByText(
            "Imported 2 startup parameters for SIRIUSe 8xAI from sirius.xml. " +
              "Skipped 3 that the Driver cannot write.",
          ),
        ).toBeTruthy();
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            OPERATION_MODE,
            EXCITATION,
          ]),
        );
      });

      it("should replace a parameter on the same object", async () => {
        const slave = await setup({
          ...SIRIUS,
          startupParameters: [FILTER, { ...OPERATION_MODE, value: 1, name: "" }],
        });
        importFile("sirius.xml", ESI);
        await screen.findByText(/^Imported 2 startup parameters/);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            FILTER,
            OPERATION_MODE,
            EXCITATION,
          ]),
        );
      });

      it("should leave out the object that holds the mode of the card", async () => {
        const slave = await setup({
          productCode: 252,
          revision: 0x00010000,
          settings: createDewesoftXSettings(false),
          inputs: [STRAIN_ENTRY],
        });
        importFile("stg.xml", ESI);
        expect(
          await screen.findByText(
            "Imported 1 startup parameter for IOLITEr 6xSTG from stg.xml. Left out " +
              "0x8020:09, which holds the mode of the device. Synnax sets the mode.",
          ),
        ).toBeTruthy();
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            {
              index: 0x8020,
              subIndex: 10,
              dataType: "uint16",
              value: 3,
              name: "Bus2 mode",
            },
          ]),
        );
      });

      it("should name the devices of a file that has none for the slave", async () => {
        await setup({ productCode: 252, revision: 0x00020000 });
        importFile("other.xml", ESI);
        expect(
          await screen.findByText(
            "The ESI file lists no device with product code 0x00FC and revision " +
              "0x00020000. It is for IOLITEr 6xSTG, SIRIUSe 8xAI.",
          ),
        ).toBeTruthy();
        expect(screen.queryByRole("textbox", { name: "Index 1" })).toBeNull();
      });

      it("should reject a file that is not XML", async () => {
        await setup(SIRIUS);
        importFile("notes.xml", "not xml <");
        expect(
          await screen.findByText("The file is not a valid ESI XML file."),
        ).toBeTruthy();
      });
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

  describe("settings", () => {
    const SAMPLE_RATE: EtherCAT.Device.StartupParameter = {
      index: 0x8020,
      subIndex: 1,
      dataType: "uint16",
      value: 12,
      name: "Sample rate",
    };

    it("should save a setting picked as a startup parameter", async () => {
      const settings = createSettingsDescription();
      const slave = await setup({ settings, startupParameters: [FILTER] });
      fireEvent.click(screen.getByRole("button", { name: "Sample rate" }));
      fireEvent.click(await screen.findByRole("option", { name: "1 kS/s" }));
      await save();
      await waitFor(async () => {
        const { properties } = await retrieve(slave.key);
        expect(properties.startupParameters).toEqual([FILTER, SAMPLE_RATE]);
        expect(properties.settings).toEqual(settings);
      });
    });

    it("should remove the startup parameter of a cleared setting", async () => {
      const slave = await setup({
        settings: createSettingsDescription(),
        startupParameters: [SAMPLE_RATE, FILTER],
      });
      fireEvent.click(screen.getByRole("button", { name: "Clear 1 kS/s" }));
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          FILTER,
        ]),
      );
    });
  });

  describe("setup", () => {
    it("should offer to switch a card that sends data only DewesoftX reads", async () => {
      await setup({
        settings: createDewesoftXSettings(true),
        inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY],
      });
      expect(await screen.findByText("Set up for DewesoftX")).toBeTruthy();
      expect(findButton("Switch to Synnax only")).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Sample rate" })).toBeNull();
    });

    it("should hide the settings of a card that DewesoftX sets up", async () => {
      await setup({ settings: createDewesoftXSettings(true), inputs: [STRAIN_ENTRY] });
      expect(await screen.findByText("Set up by DewesoftX")).toBeTruthy();
      expect(
        screen.getByText(
          "DewesoftX sets up this device, so Synnax does not change its settings.",
        ),
      ).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Sample rate" })).toBeNull();
      expect(screen.queryByText("Switch to Synnax only")).toBeNull();
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
