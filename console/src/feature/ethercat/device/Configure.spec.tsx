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
  CONTROL_ENTRY,
  createDewesoftXSettings,
  createIdentifier,
  createSettingsDescription,
  createSlaveDevice,
  STATUS_ENTRY,
  STRAIN_ENTRY,
  STRAIN_SAMPLE_ENTRY,
} from "@/feature/ethercat/testutil";
import { type Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Device } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import {
  answerCommandsUntil,
  answerNextCommand,
  type CommandAnswer,
  reportTaskRunning,
  reportTaskStopped,
} from "@/platform/task/testutil";
import { fakePickedFile, interceptFilePicker, uniqueName } from "@/testutil";

const client = createTestClient();

const onCreateReadTask = vi.fn();
const onCreateWriteTask = vi.fn();
const onBack = vi.fn();

const useConfigureWithActionsModal = Modals.create<Device.ConfigureModalParams>(
  ({ deviceKey, close }) => (
    <EtherCAT.Device.HARDWARE.Configure
      deviceKey={deviceKey}
      close={close}
      onCreateReadTask={onCreateReadTask}
      onCreateWriteTask={onCreateWriteTask}
      onBack={onBack}
    />
  ),
);

const IDENTIFIER_FORMAT_ERROR =
  "Identifier must start with a letter and contain only letters, numbers, and underscores";

const OP_MODE = { index: 0, assignActivate: 0x0300, sync0Cycle: 0, sync1Cycle: 0 };

const FILTER: ethercat.StartupParameter = {
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
  startupParameters?: ethercat.StartupParameter[];
  settings?: EtherCAT.Device.SlaveProperties["settings"];
  inputs?: ethercat.PDOEntry[];
  outputs?: ethercat.PDOEntry[];
  vendorId?: number;
  productCode?: number;
  revision?: number;
}

const VENDOR_ID = 0xdebe50f7;

const setup = async ({
  configured = true,
  useOpen = Device.useConfigureModal,
  analytics,
  startupParameters = [],
  settings,
  inputs = [],
  outputs = [],
  vendorId = VENDOR_ID,
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
      pdos: { inputs, outputs },
      vendorId,
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
    onCreateReadTask.mockClear();
    onCreateWriteTask.mockClear();
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

      const VENDOR = "#xDEBE50F7";

      const ESI = `<?xml version="1.0" encoding="utf-8"?>
      <EtherCATInfo>
        <Vendor><Id>${VENDOR}</Id></Vendor>
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
                  ${initCmd("PS", "#x8050", "1", "FFFFFFFFFFFFFFFF", "Minus one")}
                </CoE>
              </Mailbox>
            </Device>
          </Devices>
        </Descriptions>
      </EtherCATInfo>`;

      /** @returns an ESI file of the vendor with one device of each mailbox. */
      const esiOf = (mailboxes: string[], vendor = VENDOR) =>
        `<EtherCATInfo><Vendor><Id>${vendor}</Id></Vendor><Descriptions><Devices>
          ${mailboxes
            .map(
              (mailbox, i) => `<Device>
                <Type ProductCode="${0x67 + i}" RevisionNo="1">SIRIUSe ${i + 1}</Type>
                <Mailbox>${mailbox}</Mailbox>
              </Device>`,
            )
            .join("")}
        </Devices></Descriptions></EtherCATInfo>`;

      const SIRIUS = { productCode: 0x66, revision: 0x01000001 };
      const OPERATION_MODE: ethercat.StartupParameter = {
        index: 0x8020,
        subIndex: 7,
        dataType: "uint16",
        value: 6,
        name: "OperationMode",
      };
      const EXCITATION: ethercat.StartupParameter = {
        index: 0x8040,
        subIndex: 6,
        dataType: "uint32",
        value: 0x40a00000,
        name: "Excitation",
      };

      const importFile = (name: string, contents: string | Uint8Array<ArrayBuffer>) => {
        const picker = interceptFilePicker();
        fireEvent.click(findButton("Import from ESI file"));
        picker.selectFiles([fakePickedFile(name, contents)]);
      };

      afterEach(() => vi.restoreAllMocks());

      it("should add the commands that the device runs before SAFE-OP", async () => {
        const slave = await setup(SIRIUS);
        importFile("sirius.xml", ESI);
        await screen.findByText("Imported 2 startup parameters for SIRIUSe 8xAI");
        expect(
          screen.getByText("Skipped 4 that the Driver cannot write."),
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
        await screen.findByText("Imported 1 startup parameter for IOLITEr 6xSTG");
        expect(
          screen.getByText("Left out the mode of the device at 0x8020:09."),
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

      it("should say that a file lists no devices", async () => {
        await setup(SIRIUS);
        importFile("empty.xml", esiOf([]));
        expect(
          await screen.findByText(
            "The ESI file lists no device with product code 0x0066 and revision " +
              "0x01000001. It lists no devices.",
          ),
        ).toBeTruthy();
      });

      it("should keep the order of the writes to one object", async () => {
        const write = (
          subIndex: number,
          dataType: string,
          value: number,
          name: string,
        ) => ({
          index: 0x1c12,
          subIndex,
          dataType,
          value,
          name,
        });
        const slave = await setup({
          productCode: 0x67,
          revision: 1,
          startupParameters: [FILTER, write(0, "uint8", 1, "")],
        });
        importFile(
          "pdo.xml",
          esiOf([
            `<CoE>
              ${initCmd("PS", "#x1c12", "0", "00", "Clear")}
              ${initCmd("PS", "#x1c12", "1", "0016", "Assign")}
              ${initCmd("PS", "#x1c12", "0", "01", "Count")}
            </CoE>`,
          ]),
        );
        await screen.findByText(/^Imported 3 startup parameters/);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            FILTER,
            write(0, "uint8", 0, "Clear"),
            write(1, "uint16", 0x1600, "Assign"),
            write(0, "uint8", 1, "Count"),
          ]),
        );
      });

      it("should merge into the parameters as they are when the file opens", async () => {
        const slave = await setup({ ...SIRIUS, startupParameters: [FILTER] });
        const picker = interceptFilePicker();
        fireEvent.click(findButton("Import from ESI file"));
        fireEvent.click(screen.getByRole("button", { name: "Remove parameter 1" }));
        picker.selectFiles([fakePickedFile("sirius.xml", ESI)]);
        await screen.findByText(/^Imported 2 startup parameters/);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            OPERATION_MODE,
            EXCITATION,
          ]),
        );
      });

      it("should reject a file for another vendor", async () => {
        await setup(SIRIUS);
        importFile("other.xml", ESI.replace(VENDOR, "#x00000002"));
        expect(
          await screen.findByText(
            "The ESI file is not for vendor 0xDEBE50F7, the vendor of the device.",
          ),
        ).toBeTruthy();
      });

      it("should name three devices and others of a file with more", async () => {
        await setup({ productCode: 0x99, revision: 1 });
        importFile("many.xml", esiOf(["", "", "", ""]));
        expect(
          await screen.findByText(
            "The ESI file lists no device with product code 0x0099 and revision " +
              "0x00000001. It is for SIRIUSe 1, SIRIUSe 2, SIRIUSe 3, and others.",
          ),
        ).toBeTruthy();
      });

      it("should read a command on a catalog field as the type of the field", async () => {
        const slave = await setup({
          productCode: 0x67,
          revision: 1,
          settings: createSettingsDescription(),
        });
        importFile(
          "strain.xml",
          esiOf([
            `<CoE>${initCmd("PS", "#x8040", "#x1c", "0000a040", "Excitation")}</CoE>`,
          ]),
        );
        await screen.findByText(/^Imported 1 startup parameter /);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            {
              index: 0x8040,
              subIndex: 0x1c,
              dataType: "float32",
              value: 5,
              name: "Excitation",
            },
          ]),
        );
      });

      it("should read a file in the encoding that its declaration names", async () => {
        const slave = await setup({ productCode: 0x67, revision: 1 });
        const command = initCmd("PS", "#x8050", "1", "01", "Offset in °C");
        const text = `<?xml version="1.0" encoding="ISO-8859-1"?>${esiOf([
          `<CoE>${command}</CoE>`,
        ])}`;
        importFile(
          "latin.xml",
          Uint8Array.from(text, (c) => c.charCodeAt(0)),
        );
        await screen.findByText(/^Imported 1 startup parameter /);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            {
              index: 0x8050,
              subIndex: 1,
              dataType: "uint8",
              value: 1,
              name: "Offset in °C",
            },
          ]),
        );
      });

      it("should read a UTF-16 file that starts with a byte order mark", async () => {
        const slave = await setup({ productCode: 0x67, revision: 1 });
        const text = `\uFEFF${esiOf([`<CoE>${initCmd("PS", "#x8050", "1", "01", "Gain")}</CoE>`])}`;
        const bytes = new Uint8Array(
          new Uint16Array(Array.from(text, (c) => c.charCodeAt(0))).buffer,
        );
        importFile("wide.xml", bytes);
        await screen.findByText(/^Imported 1 startup parameter /);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            { index: 0x8050, subIndex: 1, dataType: "uint8", value: 1, name: "Gain" },
          ]),
        );
      });

      it("should read a UTF-8 file with a byte order mark over its declaration", async () => {
        const slave = await setup({ productCode: 0x67, revision: 1 });
        const command = initCmd("PS", "#x8050", "1", "01", "Offset in °C");
        const text = `\uFEFF<?xml version="1.0" encoding="ISO-8859-1"?>${esiOf([
          `<CoE>${command}</CoE>`,
        ])}`;
        importFile("marked.xml", new TextEncoder().encode(text));
        await screen.findByText(/^Imported 1 startup parameter /);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            {
              index: 0x8050,
              subIndex: 1,
              dataType: "uint8",
              value: 1,
              name: "Offset in °C",
            },
          ]),
        );
      });

      it("should reject a file in an encoding that Synnax cannot read", async () => {
        await setup({ productCode: 0x67, revision: 1 });
        importFile("odd.xml", `<?xml version="1.0" encoding="EBCDIC-9"?>${esiOf([])}`);
        expect(
          await screen.findByText(
            "The ESI file uses EBCDIC-9, an encoding that Synnax cannot read.",
          ),
        ).toBeTruthy();
      });

      it("should import an 8-byte value that a number holds", async () => {
        const slave = await setup({ productCode: 0x67, revision: 1 });
        importFile(
          "count.xml",
          esiOf([
            `<CoE>${initCmd("PS", "#x8050", "1", "0100000000000000", "One")}</CoE>`,
          ]),
        );
        await screen.findByText(/^Imported 1 startup parameter /);
        await save();
        await waitFor(async () =>
          expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
            { index: 0x8050, subIndex: 1, dataType: "uint64", value: 1, name: "One" },
          ]),
        );
      });

      it("should neither import nor count the commands of other mailboxes", async () => {
        await setup({ productCode: 0x67, revision: 1 });
        importFile(
          "soe.xml",
          esiOf([
            `<CoE>${initCmd("PS", "#x8050", "1", "01", "CoE")}</CoE>
            <SoE>${initCmd("PS", "#x8051", "1", "01", "SoE")}</SoE>`,
          ]),
        );
        expect(
          await screen.findByText("Imported 1 startup parameter for SIRIUSe 1"),
        ).toBeTruthy();
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
    const SAMPLE_RATE: ethercat.StartupParameter = {
      index: 0x8020,
      subIndex: 1,
      dataType: "uint16",
      value: 12,
      name: "",
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

    it("should show the value that the slave holds for a setting not set", async () => {
      const settings = createSettingsDescription();
      settings.device[0].value = 4;
      await setup({ settings, startupParameters: [SAMPLE_RATE] });
      fireEvent.click(screen.getByRole("button", { name: "Clear 1 kS/s" }));
      expect(screen.getByRole("button", { name: "Sample rate" }).textContent).toBe(
        "20 kS/s",
      );
    });

    it("should show the value that the slave holds for a number not set", async () => {
      const settings = createSettingsDescription();
      settings.device[0] = { ...settings.device[0], options: undefined, value: 250 };
      await setup({ settings });
      expect(cell("Sample rate").placeholder).toBe("250");
    });

    it("should show the shortest form of a float that the slave holds", async () => {
      const settings = createSettingsDescription();
      settings.device[0] = {
        ...settings.device[0],
        name: "Excitation voltage",
        dataType: "float32",
        options: undefined,
        value: Math.fround(3.3),
      };
      await setup({ settings });
      expect(cell("Excitation voltage").placeholder).toBe("3.3");
    });

    it("should prompt for a setting whose held value has no option", async () => {
      const settings = createSettingsDescription();
      settings.device[0].value = 0;
      await setup({ settings });
      expect(await screen.findByText("Select sample rate")).toBeTruthy();
    });

    it("should keep an acronym in the name of a setting in its placeholder", async () => {
      const settings = createSettingsDescription();
      settings.device[0].name = "Module sample rate (DC sync)";
      await setup({ settings });
      expect(
        await screen.findByText("Select module sample rate (DC sync)"),
      ).toBeTruthy();
    });

    it("should show the name of the setting for a parameter with no name", async () => {
      await setup({
        settings: createSettingsDescription(),
        startupParameters: [SAMPLE_RATE],
      });
      expect(cell("Name 1").value).toBe("Sample rate");
    });

    it("should not store the name of a setting after its address changes", async () => {
      const slave = await setup({
        settings: createSettingsDescription(),
        startupParameters: [SAMPLE_RATE],
      });
      commit("Subindex 1", "0x02");
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          { ...SAMPLE_RATE, subIndex: 2 },
        ]),
      );
    });

    it("should keep the name that the user gave the parameter of a setting", async () => {
      const named = { ...SAMPLE_RATE, value: 4, name: "Rate" };
      const slave = await setup({
        settings: createSettingsDescription(),
        startupParameters: [named],
      });
      fireEvent.click(screen.getByRole("button", { name: "Sample rate 20 kS/s" }));
      fireEvent.click(await screen.findByRole("option", { name: "1 kS/s" }));
      await save();
      await waitFor(async () =>
        expect((await retrieve(slave.key)).properties.startupParameters).toEqual([
          { ...named, value: 12 },
        ]),
      );
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

  describe("restart", () => {
    const RESTART = "Save and restart tasks";

    const createRunningRead = async (slave: EtherCAT.Device.SlaveDevice) => {
      const rack = await client.racks.retrieve({ key: slave.rack });
      const read = await rack.createTask({
        name: uniqueName("ecat_read"),
        type: EtherCAT.Task.READ_TYPE,
        config: EtherCAT.Task.READ_SCHEMAS.config.parse({
          channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
        }),
      });
      await reportTaskRunning(client, read);
      return read;
    };

    it("should stop the tasks on the network, save, then start them", async () => {
      const slave = await setup();
      const read = await createRunningRead(slave);
      const driver = await answerCommandsUntil(client, [read], "start");
      const name = uniqueName("renamed");
      fireEvent.change(nameInput(), { target: { value: name } });
      await screen.findByText(RESTART);
      await save(RESTART);
      const commands = await driver.answered;
      expect(commands.map((c) => c.type)).toEqual(["stop", "start"]);
      expect((await retrieve(slave.key)).name).toBe(name);
    });

    it("should offer no restart once no task on the network runs", async () => {
      const slave = await setup();
      const read = await createRunningRead(slave);
      const elsewhere = await createSlaveDevice(client, slave.rack, {
        network: "eth1",
      });
      await createRunningRead(elsewhere);
      expect(await screen.findByText(RESTART)).toBeTruthy();
      await reportTaskStopped(client, read);
      await waitFor(() => expect(screen.queryByText(RESTART)).toBeNull());
    });
  });

  describe("setup", () => {
    it("should offer to switch a card that sends data only DewesoftX reads", async () => {
      await setup({
        settings: createDewesoftXSettings(true),
        inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY],
      });
      expect(
        await screen.findByText("Only DewesoftX can use this device"),
      ).toBeTruthy();
      expect(findButton("Switch to Synnax")).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Sample rate" })).toBeNull();
    });

    it("should hide the settings and startup parameters of a card that DewesoftX sets up", async () => {
      await setup({
        settings: createDewesoftXSettings(true, 2),
        inputs: [STRAIN_ENTRY],
      });
      expect(await screen.findByText("DewesoftX sets up this device")).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Sample rate" })).toBeNull();
      expect(screen.queryByText("Startup parameters")).toBeNull();
      expect(findButton("Switch to Synnax")).toBeTruthy();
    });

    it("should open the setup docs from the note of a card that DewesoftX sets up", async () => {
      const open = vi.spyOn(window, "open").mockReturnValue(null);
      await setup({
        settings: createDewesoftXSettings(true, 2),
        inputs: [STRAIN_ENTRY],
      });
      await screen.findByText("DewesoftX sets up this device");
      fireEvent.click(findButton("What does this mean?"));
      expect(open).toHaveBeenCalledWith(
        "https://docs.synnaxlabs.com/reference/hardware/ethercat/dewesoft#set-up-a-module-in-synnax",
        "_blank",
        "noopener,noreferrer",
      );
      open.mockRestore();
    });

    it("should ask for bus 1 for a standalone card on bus 2", async () => {
      await setup({
        settings: createDewesoftXSettings(false, 2),
        inputs: [STRAIN_ENTRY],
      });
      expect(
        await screen.findByText("Connect Synnax to EtherCAT bus 1 IN"),
      ).toBeTruthy();
      expect(screen.queryByText("Switch to Synnax")).toBeNull();
    });

    /** Opens the card, by default one only DewesoftX can use, and clicks the switch. */
    const clickSwitch = async (
      answer: Omit<CommandAnswer, "type"> = {},
      card: Pick<SetupArgs, "settings" | "inputs"> = {
        settings: createDewesoftXSettings(true),
        inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY],
      },
    ) => {
      const slave = await setup(card);
      const rack = await client.racks.retrieve({ key: slave.rack });
      const scan = await rack.createTask({
        name: uniqueName("ecat_scan"),
        type: EtherCAT.Task.SCAN_TYPE,
        config: {},
      });
      const driver = await answerNextCommand(client, scan, {
        ...answer,
        type: "set_standalone",
      });
      fireEvent.click(findButton("Switch to Synnax"));
      expect(await driver.answered).toMatchObject({ args: { device: slave.key } });
    };

    it("should ask for a power restart once the card switches", async () => {
      await clickSwitch();
      expect(await screen.findByText("Restart the power of the device")).toBeTruthy();
      expect(screen.queryByText("Switch to Synnax")).toBeNull();
    });

    it("should show why the card did not switch", async () => {
      await clickSwitch({ variant: "error", message: "Slave 3 refused the switch" });
      expect(await screen.findByText(/Slave 3 refused the switch/)).toBeTruthy();
      expect(screen.queryByText("Restart the power of the device")).toBeNull();
    });

    it("should ask for a power restart for a switched card that sends DewesoftX data", async () => {
      await setup({
        settings: createDewesoftXSettings(false),
        inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY],
      });
      expect(await screen.findByText("Restart the power of the device")).toBeTruthy();
      expect(screen.queryByText("Switch to Synnax")).toBeNull();
    });

    it("should offer no switch for a card whose mode the scan did not read", async () => {
      await setup({
        settings: { ...createDewesoftXSettings(true), mode: undefined },
        inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY],
      });
      expect(
        await screen.findByText("Only DewesoftX can use this device"),
      ).toBeTruthy();
      expect(screen.queryByText("Switch to Synnax")).toBeNull();
    });

    it("should ask for bus 1 and a power restart once a card on bus 2 switches", async () => {
      await clickSwitch(
        {},
        { settings: createDewesoftXSettings(true, 2), inputs: [STRAIN_ENTRY] },
      );
      expect(await screen.findByText("Restart the power of the device")).toBeTruthy();
      expect(
        screen.getByText("First, connect Synnax to EtherCAT bus 1 IN."),
      ).toBeTruthy();
    });
  });

  describe("actions", () => {
    it("should create a read task for a new slave only after it saves", async () => {
      const slave = await setup({
        configured: false,
        useOpen: useConfigureWithActionsModal,
        inputs: [STRAIN_ENTRY],
      });
      fireEvent.change(identifierInput(), { target: { value: "slave_1" } });
      await save("Save and create read task");
      await waitFor(() => expect(onCreateReadTask).toHaveBeenCalledWith(slave.key));
      expect((await retrieve(slave.key)).configured).toBe(true);
    });

    it("should create a write task for a new slave with outputs", async () => {
      const slave = await setup({
        configured: false,
        useOpen: useConfigureWithActionsModal,
        outputs: [CONTROL_ENTRY],
      });
      expect(screen.queryByText("Save and create read task")).toBeNull();
      fireEvent.change(identifierInput(), { target: { value: "slave_1" } });
      await save("Save and create write task");
      await waitFor(() => expect(onCreateWriteTask).toHaveBeenCalledWith(slave.key));
      expect(onCreateReadTask).not.toHaveBeenCalled();
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
