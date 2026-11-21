// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { Value } from "@/vis/value";

const schema = z.object({ precision: z.number().optional() }).loose();
const path = deep.path<z.infer<typeof schema>>();

const typePrecision = (value: string): void => {
  const input = screen
    .getByText("Precision")
    .closest(".lyra-input__item")
    ?.querySelector("input");
  if (input == null) throw new Error("expected a precision input");
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const renderTelemForm = () =>
  renderForm(
    <Value.TelemForm />,
    { precision: 2 },
    schema,
    createSynnaxWrapper({ client: null }),
  );

describe("Value.TelemForm", () => {
  it("should round a typed fractional precision to a whole number", () => {
    const { value } = renderTelemForm();
    typePrecision("4.6");
    expect(value(path.precision)).toBe(5);
  });

  it("should clamp a typed precision to 17", () => {
    const { value } = renderTelemForm();
    typePrecision("40");
    expect(value(path.precision)).toBe(17);
  });
});

describe("Value.TelemForm channel", () => {
  const client = createTestClient();
  const valuePath = deep.path<schematic.ValueNodeConfig>();
  let wrapper: FC<PropsWithChildren>;
  let ch: channel.Channel;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
  });

  it("should store 0 when the channel is cleared", async () => {
    const { findByLabelText, value } = renderForm(
      <Value.TelemForm />,
      schematic.valueNodeConfigZ.parse({ variant: "value", channel: ch.key }),
      schematic.valueNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value(valuePath.channel)).toBe(0));
  });
});
