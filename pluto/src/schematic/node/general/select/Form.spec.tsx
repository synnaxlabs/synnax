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

import { SelectForm } from "@/schematic/node/general/select/Form";
import { createConfig } from "@/schematic/node/registry";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const path = deep.path<schematic.SelectNodeConfig>();

const client = createTestClient();

describe("SelectForm", () => {
  // Without a box, the dropdown's virtual list shows no items.
  beforeAll(() => mockGeometry(100, 100));
  let wrapper: FC<PropsWithChildren>;
  let ch: channel.Channel;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.UINT8,
      virtual: true,
    });
  });

  it("should edit the options only while no enumeration is picked", async () => {
    const created = await client.enumerations.create({
      name: `select-${id.create()}`,
      entries: [],
    });
    const { findByRole, findByLabelText, getByPlaceholderText, queryByRole, value } =
      renderForm(
        <SelectForm tab="options" onTabChange={vi.fn()} />,
        createConfig({ variant: "select" }),
        schematic.selectNodeConfigZ,
        wrapper,
      );
    const add = { name: "Add option" };
    expect(await findByRole("button", add)).toBeTruthy();
    fireEvent.click(await findByRole("button", { name: "Enumeration" }));
    fireEvent.change(getByPlaceholderText("Search enumerations..."), {
      target: { value: created.name },
    });
    fireEvent.click(await findByRole("option", { name: created.name }));
    await waitFor(() => expect(value(path.enumeration)).toBe(created.key));
    const clear = await findByLabelText(`Clear ${created.name}`);
    expect(queryByRole("button", add)).toBeNull();
    fireEvent.click(clear);
    await waitFor(() => expect(value(path.enumeration)).toBeUndefined());
    expect(await findByRole("button", add)).toBeTruthy();
  });

  it("should unbind and disable the command channel when it is cleared", async () => {
    const config = createConfig({ variant: "select", commandChannel: ch.key });
    const { findByLabelText, value } = renderForm(
      <SelectForm tab="control" onTabChange={vi.fn()} />,
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
