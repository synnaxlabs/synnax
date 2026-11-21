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
import { mockGeometry } from "@synnaxlabs/lyra/testutil";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { StateIndicator } from "@/schematic/node/general/stateIndicator";
import { CreatePromptStub } from "@/schematic/node/testutil";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.StateIndicatorNodeConfig>();

const client = createTestClient();

describe("StateIndicator form", () => {
  // Without a box, the dropdown's virtual list shows no items.
  beforeAll(() => mockGeometry(100, 100));
  let wrapper: FC<PropsWithChildren>;
  let ch: channel.Channel;
  beforeAll(async () => {
    const Base = await createAsyncSynnaxWrapper({ client });
    wrapper = ({ children }) => (
      <Base>
        <CreatePromptStub>{children}</CreatePromptStub>
      </Base>
    );
    ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
  });

  it("should store a picked enumeration", async () => {
    const created = await client.enumerations.create({
      name: `indicator-${id.create()}`,
      entries: [],
    });
    const { findByRole, getByPlaceholderText, value } = renderForm(
      <StateIndicator.spec.Form tab="options" onTabChange={vi.fn()} />,
      schematic.stateIndicatorNodeConfigZ.parse({ variant: "state_indicator" }),
      schematic.stateIndicatorNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByRole("button", { name: "Enumeration" }));
    fireEvent.change(getByPlaceholderText("Search enumerations..."), {
      target: { value: created.name },
    });
    fireEvent.click(await findByRole("option", { name: created.name }));
    await waitFor(() => expect(value(path.enumeration)).toBe(created.key));
  });

  it("should store 0 when the channel is cleared", async () => {
    const { findByLabelText, value } = renderForm(
      <StateIndicator.spec.Form tab="telemetry" onTabChange={vi.fn()} />,
      schematic.stateIndicatorNodeConfigZ.parse({
        variant: "state_indicator",
        channel: ch.key,
      }),
      schematic.stateIndicatorNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => expect(value(path.channel)).toBe(0));
  });
});
