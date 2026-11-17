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
import { type FC, type PropsWithChildren } from "react";
import { assert, beforeAll, describe, expect, it, vi } from "vitest";

import { Light } from "@/schematic/node/general/light";
import { createAsyncSynnaxWrapper, createSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.LightNodeConfig>();

const client = createTestClient();

const inputValue = (el: HTMLElement): string => {
  assert(el instanceof HTMLInputElement);
  return el.value;
};

describe("Light threshold", () => {
  it("should show 0.9 and 1.1 when the threshold is absent", () => {
    const { getByLabelText } = renderForm(
      <Light.spec.Form tab="telemetry" onTabChange={vi.fn()} />,
      schematic.lightNodeConfigZ.parse({ variant: "light" }),
      schematic.lightNodeConfigZ,
      createSynnaxWrapper({ client: null }),
    );
    expect(inputValue(getByLabelText("Lower threshold"))).toBe("0.9");
    expect(inputValue(getByLabelText("Upper threshold"))).toBe("1.1");
  });
});

describe("Light form", () => {
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
      <Light.spec.Form tab="telemetry" onTabChange={vi.fn()} />,
      schematic.lightNodeConfigZ.parse({ variant: "light", channel: ch.key }),
      schematic.lightNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value(path.channel)).toBe(0));
  });
});
