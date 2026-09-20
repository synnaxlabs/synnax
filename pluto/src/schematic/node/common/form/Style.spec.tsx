// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type ReactElement, useImperativeHandle } from "react";
import { assert, describe, expect, it } from "vitest";
import { z } from "zod";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const schema = z.object({
  label: Label.configZ,
  color: color.crudeZ.optional(),
  scale: z.number(),
  orientation: z.string(),
  specKey: z.string().optional(),
  stateOverrides: z.array(z.unknown()).optional(),
});
type Values = z.infer<typeof schema>;

const VALUES: Values = {
  label: Label.defaultConfig("Valve"),
  scale: 1,
  orientation: "left",
};

interface Handle {
  get: () => color.Crude | undefined;
}

interface HostProps {
  values: Values;
  ref?: React.Ref<Handle>;
}

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const Host = ({ values, ref }: HostProps): ReactElement => {
  const methods = Base.use({ values, schema });
  useImperativeHandle(ref, () => ({
    get: () => methods.get<color.Crude>("color", { optional: true })?.value,
  }));
  return (
    <SynnaxWrapper>
      <Base.Form<typeof schema> {...methods}>
        <Form.StyleForm />
      </Base.Form>
    </SynnaxWrapper>
  );
};

describe("Form.StyleForm", () => {
  it("should seed a zero color when the config has none", () => {
    const ref: { current: Handle | null } = { current: null };
    const { getByText } = render(<Host values={VALUES} ref={ref} />);
    expect(getByText("Color")).toBeDefined();
    assert(ref.current != null);
    expect(ref.current.get()).toEqual(color.ZERO);
  });

  it("should hide the color field when the config carries state overrides", () => {
    const { queryByText } = render(
      <Host values={{ ...VALUES, specKey: "", stateOverrides: [] }} />,
    );
    expect(queryByText("Color")).toBeNull();
  });
});
