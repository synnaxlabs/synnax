// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { z } from "zod";

export const thermocoupleTypeZ = z.enum(["J", "K", "N", "R", "S", "T", "B", "E"]);

export const ThermocoupleTypeField = Form.buildDropdownButtonSelectField({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple Type" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    hideColumnHeader: true,
    data: [
      { key: "B", name: "B" },
      { key: "E", name: "E" },
      { key: "J", name: "J" },
      { key: "K", name: "K" },
      { key: "N", name: "N" },
      { key: "R", name: "R" },
      { key: "S", name: "S" },
      { key: "T", name: "T" },
    ],
  },
});

export const labJackThermocoupleTypeZ = thermocoupleTypeZ.or(z.literal("C"));
export const LabJackThermocoupleTypeField = Form.buildDropdownButtonSelectField({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple Type" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    hideColumnHeader: true,
    data: [
      { key: "B", name: "B" },
      { key: "E", name: "E" },
      { key: "J", name: "J" },
      { key: "K", name: "K" },
      { key: "N", name: "N" },
      { key: "R", name: "R" },
      { key: "S", name: "S" },
      { key: "T", name: "T" },
      { key: "C", name: "C" },
    ],
  },
});
