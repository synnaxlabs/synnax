// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { renderThemed, treatment } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/common/primitive";
import { Input } from "@/schematic/node/general/input";
import { Primitive as Select } from "@/schematic/node/general/select/primitive";
import { Setpoint } from "@/schematic/node/general/setpoint";

describe("symbol field fonts", () => {
  it("should draw an input field value with tabular digits", async () => {
    const { getByRole } = await renderThemed(
      <Input.spec.Preview
        {...schematic.inputNodeConfigZ.parse({ variant: "input" })}
      />,
    );
    expect(treatment(getByRole("textbox").element())).toBe("numeric");
  });

  it("should draw a setpoint field value with tabular digits", async () => {
    const { getByRole } = await renderThemed(
      <Setpoint.spec.Preview
        {...schematic.setpointNodeConfigZ.parse({ variant: "setpoint" })}
      />,
    );
    expect(treatment(getByRole("textbox").element())).toBe("numeric");
  });

  it("should draw a select label with tabular digits", async () => {
    const { getByText } = await renderThemed(
      <Select.Select
        value="a"
        options={[{ key: "a", name: "Stage 2", value: 2 }]}
        onSend={() => {}}
      />,
    );
    expect(treatment(getByText("Stage 2").element())).toBe("numeric");
  });

  it("should draw a readout value with tabular digits", async () => {
    const { getByText } = await renderThemed(
      <Primitive.Readout size="medium" value="100.0" />,
    );
    expect(treatment(getByText("100.0").element())).toBe("numeric");
  });
});
