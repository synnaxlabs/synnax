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

const { onlyReaderOf, setupOf } = EtherCAT.Device.Setup;

const props = (
  settings: EtherCAT.Device.SlaveProperties["settings"],
  inputs = [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY],
): EtherCAT.Device.SlaveProperties => ({
  ...EtherCAT.Device.ZERO_SLAVE_PROPERTIES,
  pdos: { inputs, outputs: [] },
  settings,
});

describe("setupOf", () => {
  it("should find a card without marked entries set up for Synnax", () => {
    expect(setupOf(props(createSettingsDescription()))).toEqual({ kind: "synnax" });
  });

  it("should find a shared card that sends DewesoftX data set up for DewesoftX", () => {
    expect(setupOf(props(createDewesoftXSettings(true)))).toEqual({
      kind: "other",
      reader: "DewesoftX",
    });
  });

  it("should ask for a restart of a switched card that still sends DewesoftX data", () => {
    expect(setupOf(props(createDewesoftXSettings(false)))).toEqual({
      kind: "restart",
      reader: "DewesoftX",
    });
  });

  it("should find a shared card on the port that DewesoftX does not use shared", () => {
    expect(setupOf(props(createDewesoftXSettings(true), [STRAIN_ENTRY]))).toEqual({
      kind: "shared",
      reader: "DewesoftX",
    });
  });

  it("should find a standalone card that sends Synnax data set up for Synnax", () => {
    expect(setupOf(props(createDewesoftXSettings(false), [STRAIN_ENTRY]))).toEqual({
      kind: "synnax",
    });
  });
});

describe("onlyReaderOf", () => {
  it("should name the program that alone reads an entry, and none for others", () => {
    const card = props(createDewesoftXSettings(true));
    expect(onlyReaderOf(card, STRAIN_SAMPLE_ENTRY)).toEqual("DewesoftX");
    expect(onlyReaderOf(card, STRAIN_ENTRY)).toBeNull();
  });
});
