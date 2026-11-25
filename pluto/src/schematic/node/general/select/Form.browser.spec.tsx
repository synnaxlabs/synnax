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
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Node } from "@/schematic/node";
import { Select } from "@/schematic/node/general/select";
import { CreatePromptStub } from "@/schematic/node/testutil";
import { createAsyncSynnaxWrapper, pickOption, renderForm } from "@/testutil";

const path = deep.path<schematic.SelectNodeConfig>();

const client = createTestClient();

describe("SelectForm", () => {
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
      dataType: DataType.UINT8,
      virtual: true,
    });
  });

  it("should show the entries of a picked enumeration only", async () => {
    const created = await client.enumerations.create({
      name: `select-${id.create()}`,
      entries: [{ key: id.create(), name: "Primed", value: 0, abnormal: false }],
    });
    const { findByRole, findByLabelText, queryByLabelText, value } = renderForm(
      <Select.spec.Form tab="options" onTabChange={vi.fn()} />,
      Node.createConfig({ variant: "select" }),
      schematic.selectNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByRole("button", { name: "Enumeration" }));
    expect(queryByLabelText("Name 1")).toBeNull();
    await pickOption(created.name, "Search enumerations...");
    await waitFor(() => expect(value(path.enumeration)).toBe(created.key));
    const name = await screen.findByLabelText<HTMLInputElement>("Name 1");
    expect(name.value).toBe("Primed");
    fireEvent.click(await findByLabelText(`Clear ${created.name}`));
    await waitFor(() => expect(value(path.enumeration)).toBeUndefined());
    await waitFor(() => expect(queryByLabelText("Name 1")).toBeNull());
  });

  it("should unbind and disable the command channel when it is cleared", async () => {
    const config = Node.createConfig({ variant: "select", commandChannel: ch.key });
    const { findByLabelText, value } = renderForm(
      <Select.spec.Form tab="control" onTabChange={vi.fn()} />,
      config,
      schematic.selectNodeConfigZ,
      wrapper,
    );
    fireEvent.click(await findByLabelText(`Clear ${ch.name}`));
    await waitFor(() => {
      expect(value(path.commandChannel)).toBe(0);
      expect(value(path.disabled)).toBe(true);
    });
  });
});
