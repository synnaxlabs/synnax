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
import { fireEvent, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { assert, beforeAll, describe, expect, it } from "vitest";

import { Box } from "@/schematic/node/general/box";
import { Tank } from "@/schematic/node/vessels/tank";
import { createAsyncSynnaxWrapper, createSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.TankNodeConfig>();

const createTank = (channel: channel.Key): schematic.TankNodeConfig =>
  schematic.tankNodeConfigZ.parse({ variant: "tank", channel });

const renderFill = (
  config: schematic.TankNodeConfig,
  wrapper: FC<PropsWithChildren>,
) => {
  const result = renderForm(
    <Tank.spec.Form />,
    config,
    schematic.tankNodeConfigZ,
    wrapper,
  );
  fireEvent.click(result.getByRole("tab", { name: "Fill" }));
  return result;
};

const inputValue = (el: HTMLElement): string => {
  assert(el instanceof HTMLInputElement);
  return el.value;
};

describe("Tank style form", () => {
  const wrapper = createSynnaxWrapper({ client: null });

  it("should show the 50 by 10 radius when the tank radius is absent", () => {
    const { getByLabelText } = renderForm(
      <Tank.spec.Form />,
      schematic.tankNodeConfigZ.parse({ variant: "tank" }),
      schematic.tankNodeConfigZ,
      wrapper,
    );
    expect(inputValue(getByLabelText("X border radius"))).toBe("50");
    expect(inputValue(getByLabelText("Y border radius"))).toBe("10");
  });

  it("should show a 3 px radius when the box radius is absent", () => {
    const { getByLabelText } = renderForm(
      <Box.spec.Form />,
      schematic.boxNodeConfigZ.parse({ variant: "box" }),
      schematic.boxNodeConfigZ,
      wrapper,
    );
    expect(inputValue(getByLabelText("Border radius"))).toBe("3");
  });
});

describe("Tank fill form", () => {
  const wrapper = createSynnaxWrapper({ client: null });

  it("should hide the display fields when the tank reads no channel", () => {
    const { queryByRole } = renderFill(createTank(0), wrapper);
    expect(queryByRole("checkbox", { name: "Value" })).toBeNull();
  });

  it("should show the display fields when the tank reads a channel", () => {
    const { getByRole } = renderFill(createTank(7), wrapper);
    expect(getByRole("checkbox", { name: "Value" })).toBeTruthy();
  });
});

describe("Tank fill channel", () => {
  const client = createTestClient();
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
    const { findByLabelText, value } = renderFill(createTank(ch.key), wrapper);
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value(path.channel)).toBe(0));
  });
});
