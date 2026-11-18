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
import { beforeAll, describe, expect, it } from "vitest";

import { Toggle } from "@/schematic/node/common/toggle";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.ValveNodeConfig>();

const client = createTestClient();

describe("Toggle.ChannelForm", () => {
  let wrapper: FC<PropsWithChildren>;
  let state: channel.Channel;
  let command: channel.Channel;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    [state, command] = await client.channels.create([
      { name: id.create(), dataType: DataType.UINT8, virtual: true },
      { name: id.create(), dataType: DataType.UINT8, virtual: true },
    ]);
  });

  const renderChannels = () =>
    renderForm(
      <Toggle.ChannelForm />,
      schematic.valveNodeConfigZ.parse({
        variant: "valve",
        stateChannel: state.key,
        commandChannel: command.key,
      }),
      schematic.valveNodeConfigZ,
      wrapper,
    );

  it("should store 0 when the state channel is cleared", async () => {
    const { findByLabelText, value } = renderChannels();
    fireEvent.click(await findByLabelText(`Clear ${state.name}`));
    await waitFor(() => expect(value(path.stateChannel)).toBe(0));
    expect(value(path.commandChannel)).toBe(command.key);
  });

  it("should store 0 when the command channel is cleared", async () => {
    const { findByLabelText, value } = renderChannels();
    fireEvent.click(await findByLabelText(`Clear ${command.name}`));
    await waitFor(() => expect(value(path.commandChannel)).toBe(0));
    expect(value(path.stateChannel)).toBe(state.key);
  });
});
