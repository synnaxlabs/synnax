// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createDewesoftXSettings,
  createSettingsDescription,
  STRAIN_ENTRY,
  STRAIN_SAMPLE_ENTRY,
} from "@/feature/ethercat/testutil";

const { canSwitch, of, onlyReaderOf } = EtherCAT.Device.Setup;

const props = (
  settings: EtherCAT.Device.SlaveProperties["settings"],
  inputs = [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY],
): EtherCAT.Device.SlaveProperties => ({
  ...EtherCAT.Device.ZERO_SLAVE_PROPERTIES,
  pdos: { inputs, outputs: [] },
  settings,
});

/** A card with one bus whose held mode sends data that only DewesoftX reads. */
const SINGLE_BUS_SETTINGS: EtherCAT.Device.SlaveProperties["settings"] = {
  ...createSettingsDescription(),
  mode: {
    index: 0x8020,
    subIndex: 7,
    value: 1,
    shared: true,
    onlyReadBy: "DewesoftX",
    switch: [{ index: 0x8020, subIndex: 7, dataType: "uint16", value: 0, name: "" }],
  },
};

type Settings = NonNullable<EtherCAT.Device.SlaveProperties["settings"]>;

/** The mode a Driver gave before it named the software and the bus. */
const withoutNames = (settings: Settings): Settings =>
  settings.mode == null
    ? settings
    : {
        ...settings,
        mode: { ...settings.mode, sharedWith: undefined, bus: undefined },
      };

describe("of", () => {
  it("should find a card without marked entries set up for Synnax", () => {
    expect(of(props(createSettingsDescription()))).toEqual({ kind: "synnax" });
  });

  it("should find a dual card that sends DewesoftX data set up for DewesoftX", () => {
    expect(of(props(createDewesoftXSettings(true)))).toEqual({
      kind: "other",
      reader: "DewesoftX",
    });
  });

  it("should find a card whose held mode only DewesoftX reads set up for it", () => {
    expect(of(props(SINGLE_BUS_SETTINGS, [STRAIN_ENTRY]))).toEqual({
      kind: "other",
      reader: "DewesoftX",
    });
  });

  it("should find a standalone card that sends DewesoftX data switched", () => {
    expect(of(props(createDewesoftXSettings(false)))).toEqual({
      kind: "switched",
      reader: "DewesoftX",
    });
  });

  it("should find a card with an unread mode that sends DewesoftX data set up for it", () => {
    expect(of(props({ ...createDewesoftXSettings(true), mode: undefined }))).toEqual({
      kind: "other",
      reader: "DewesoftX",
    });
  });

  it("should find a dual card on bus 1 that sends no DewesoftX data shared", () => {
    expect(of(props(createDewesoftXSettings(true), [STRAIN_ENTRY]))).toEqual({
      kind: "shared",
      reader: "DewesoftX",
    });
  });

  it("should find a dual card on bus 2 shared", () => {
    expect(of(props(createDewesoftXSettings(true, 2), [STRAIN_ENTRY]))).toEqual({
      kind: "shared",
      reader: "DewesoftX",
    });
  });

  it("should find a standalone card on bus 2 on the wrong bus", () => {
    expect(of(props(createDewesoftXSettings(false, 2), [STRAIN_ENTRY]))).toEqual({
      kind: "bus",
    });
  });

  it("should find a standalone card on bus 1 set up for Synnax", () => {
    expect(of(props(createDewesoftXSettings(false), [STRAIN_ENTRY]))).toEqual({
      kind: "synnax",
    });
  });

  it("should name the reader of the marked entries for a mode without names", () => {
    const settings = withoutNames(createDewesoftXSettings(true));
    expect(of(props(settings))).toEqual({ kind: "other", reader: "DewesoftX" });
    expect(of(props(settings, [STRAIN_ENTRY]))).toEqual({
      kind: "shared",
      reader: "DewesoftX",
    });
  });

  it("should name other software for a mode without names or marked entries", () => {
    const settings = withoutNames(createDewesoftXSettings(true));
    expect(of(props({ ...settings, channels: [] }, [STRAIN_ENTRY]))).toEqual({
      kind: "shared",
      reader: "other software",
    });
  });
});

describe("canSwitch", () => {
  it("should allow a switch only when the catalog gives one", () => {
    expect(canSwitch(props(createDewesoftXSettings(true)))).toBe(true);
    expect(canSwitch(props(createDewesoftXSettings(false)))).toBe(false);
    expect(canSwitch(props(createSettingsDescription()))).toBe(false);
  });
});

describe("onlyReaderOf", () => {
  it("should name the program that alone reads an entry, and none for others", () => {
    const card = props(createDewesoftXSettings(true));
    expect(onlyReaderOf(card, STRAIN_SAMPLE_ENTRY)).toEqual("DewesoftX");
    expect(onlyReaderOf(card, STRAIN_ENTRY)).toBeNull();
  });

  it("should name the program that alone reads each entry in the held mode", () => {
    const card = props(SINGLE_BUS_SETTINGS, [STRAIN_ENTRY]);
    expect(onlyReaderOf(card, STRAIN_ENTRY)).toEqual("DewesoftX");
  });
});
