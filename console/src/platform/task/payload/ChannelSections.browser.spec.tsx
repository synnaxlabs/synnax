// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, selectFromDropdown } from "@/platform/task/testutil";

interface Channel {
  channel: number;
  dataType: string;
  pointer: string;
  jsonType: "string" | "number" | "boolean";
  enumValues: { label: string; value: number }[];
}

const path = deep.path<{ config: { channel: Channel } }>();

const render = async (jsonType: Channel["jsonType"]) =>
  await renderInTaskForm(
    <Task.Payload.ChannelSections
      path={path.config.channel}
      pointerPlaceholder="/value"
    />,
    {
      values: {
        config: {
          channel: {
            channel: 0,
            name: "",
            pointer: "",
            jsonType,
            dataType: "float64",
            enumValues: [{ label: "ON", value: 1 }],
          },
        },
      },
    },
  );

describe("ChannelSections", () => {
  it("should clear the enum mapping when the JSON type leaves string", async () => {
    const { form } = await render("string");
    await screen.findByText("Enum mapping");
    await selectFromDropdown("String", "Number");
    await waitFor(() => expect(screen.queryByText("Enum mapping")).toBeNull());
    expect(form.current?.get(path.config.channel.enumValues).value).toEqual([]);
  });

  it("should select the data type of a channel that configure creates", async () => {
    await render("number");
    await screen.findByText("Data type");
    expect(screen.queryByText("Set on the channel")).toBeNull();
    expect(screen.getByPlaceholderText("/value")).toBeTruthy();
    expect(screen.queryByText("Time format")).toBeNull();
  });
});
