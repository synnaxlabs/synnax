// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  mergeStartupParameters,
  parseStartupCommands,
} from "@/feature/ethercat/device/esi";

const initCmd = (
  transition: string,
  index: string,
  subIndex: string,
  data: string,
  comment: string,
  attrs = "",
) => `<InitCmd${attrs}>
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
      <Device Physics="KK">
        <Type ProductCode="252" RevisionNo="#x00010000">IOLITEr 6xSTG</Type>
        <Mailbox DataLinkLayer="true">
          <CoE SdoInfo="true">
            ${initCmd("IP", "32800", "9", "0100", "Bus1 mode")}
            ${initCmd("IP", "32800", "10", "0300", "Bus2 mode")}
          </CoE>
        </Mailbox>
      </Device>
      <Device Physics="KK">
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

describe("parseStartupCommands", () => {
  it("should read the CoE startup commands of the matching device", () => {
    expect(
      parseStartupCommands(ESI, { productCode: 252, revision: 0x00010000 }),
    ).toEqual({
      device: "IOLITEr 6xSTG",
      parameters: [
        { index: 0x8020, subIndex: 9, dataType: "uint16", value: 1, name: "Bus1 mode" },
        {
          index: 0x8020,
          subIndex: 10,
          dataType: "uint16",
          value: 3,
          name: "Bus2 mode",
        },
      ],
      skipped: 0,
    });
  });

  it("should read hexadecimal numbers and little-endian data of each width", () => {
    const { parameters } = parseStartupCommands(ESI, {
      productCode: 0x66,
      revision: 0x01000001,
    });
    expect(parameters).toEqual([
      {
        index: 0x8020,
        subIndex: 7,
        dataType: "uint16",
        value: 6,
        name: "OperationMode",
      },
      {
        index: 0x8040,
        subIndex: 6,
        dataType: "uint32",
        value: 0x40a00000,
        name: "Excitation",
      },
    ]);
  });

  it("should skip late, odd-width, and complete access commands", () => {
    expect(
      parseStartupCommands(ESI, { productCode: 0x66, revision: 0x01000001 }).skipped,
    ).toBe(3);
  });

  it("should throw when no device has the slave's product code and revision", () => {
    expect(() =>
      parseStartupCommands(ESI, { productCode: 252, revision: 0x00020000 }),
    ).toThrow(
      "The ESI file lists no device with product code 0x00FC and revision " +
        "0x00020000. It is for IOLITEr 6xSTG, SIRIUSe 8xAI.",
    );
  });

  it("should throw when the file is not XML", () => {
    expect(() =>
      parseStartupCommands("not xml <", { productCode: 252, revision: 0 }),
    ).toThrow("The file is not a valid ESI XML file.");
  });
});

describe("mergeStartupParameters", () => {
  it("should replace a parameter with the same object and add the others", () => {
    const filter = {
      index: 0x8040,
      subIndex: 4,
      dataType: "uint16",
      value: 1,
      name: "",
    };
    const mode = { index: 0x8020, subIndex: 9, dataType: "uint16", value: 1, name: "" };
    const newMode = { ...mode, value: 3, name: "Bus1 mode" };
    const bus2 = { ...mode, subIndex: 10, value: 3, name: "Bus2 mode" };
    expect(mergeStartupParameters([filter, mode], [newMode, bus2])).toEqual([
      filter,
      newMode,
      bus2,
    ]);
  });
});
