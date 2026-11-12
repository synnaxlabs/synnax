// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  renderInTaskFormWithClient,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const path = deep.path<{ config: { dataType: string } }>();

const render = async (channel: number) =>
  await renderInTaskFormWithClient(
    <Task.Payload.DataTypeField path={path.config.dataType} channel={channel} />,
    { client, values: { config: { dataType: "float64" } } },
  );

const createChannel = async () =>
  await client.channels.create({
    name: uniqueName("data_type"),
    dataType: DataType.INT32,
    virtual: true,
  });

describe("DataTypeField", () => {
  it("should lock the data type to that of a channel that exists", async () => {
    const ch = await createChannel();
    await render(ch.key);
    await screen.findByText("int32");
    expect(screen.getByText("Set on the channel")).toBeTruthy();
  });

  it("should let a channel that no longer exists take a new data type", async () => {
    const ch = await createChannel();
    await client.channels.delete(ch.key);
    const { form } = await render(ch.key);
    await screen.findByText("float64");
    expect(screen.queryByText("Set on the channel")).toBeNull();
    await selectFromDropdown("float64", "uint8");
    await waitFor(() =>
      expect(form.current?.get(path.config.dataType).value).toBe("uint8"),
    );
  });
});

describe("ReadDataTypeField", () => {
  it("should lock the data type to that of a channel that exists", async () => {
    const ch = await createChannel();
    await renderInTaskFormWithClient(
      <Task.Payload.ReadDataTypeField
        value="float64"
        onChange={() => {}}
        channel={ch.key}
      />,
      { client, values: {} },
    );
    await screen.findByText("int32");
    expect(screen.getByText("Set on the channel")).toBeTruthy();
  });
});
