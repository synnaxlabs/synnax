// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { deep } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Enumeration } from "@/enumeration";
import { Form } from "@/schematic/node/common/form";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const path = deep.path<schematic.StateIndicatorNodeConfig>();

const States = (): ReactElement => {
  const methods = Base.use({
    values: schematic.stateIndicatorNodeConfigZ.parse({
      variant: "state_indicator",
      options: [
        { key: "off", name: "Off", value: 0 },
        { key: "on", name: "On", value: 1 },
      ],
    }),
    schema: schematic.stateIndicatorNodeConfigZ,
  });
  return (
    <Base.Form<typeof schematic.stateIndicatorNodeConfigZ> {...methods}>
      <Form.States
        enumeration={path.enumeration}
        options={path.options}
        label={path.label.label}
        symbolName="State indicator"
      />
    </Base.Form>
  );
};

describe("Form.States", () => {
  it("should span the picker across the columns of the entries", async () => {
    const Wrapper = createSynnaxWrapper({ client: null });
    await renderThemed(
      <Wrapper>
        <Enumeration.CreatePromptProvider value={vi.fn()}>
          <States />
        </Enumeration.CreatePromptProvider>
      </Wrapper>,
    );
    const picker = screen
      .getByRole("button", { name: /^Enumeration/ })
      .getBoundingClientRect();
    const columns = screen
      .getAllByRole("columnheader")
      .filter((th) => th.textContent !== "");
    const first = columns[0].getBoundingClientRect();
    const last = columns[columns.length - 1].getBoundingClientRect();
    expect(picker.left).toBeCloseTo(first.left, 0);
    expect(picker.right).toBeCloseTo(last.right, 0);
  });
});
