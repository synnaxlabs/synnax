// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, selectFromDropdown } from "@/platform/task/testutil";

const PATH = "config.channel";

const render = async (jsonType: string) =>
  await renderInTaskForm(
    <Task.Payload.ChannelSections path={PATH} pointerPlaceholder="/value" />,
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
    expect(form.current?.get(`${PATH}.enumValues`).value).toEqual([]);
  });

  it("should select the data type of a channel that configure creates", async () => {
    await render("number");
    await screen.findByText("Data type");
    expect(screen.queryByText("Set on the channel")).toBeNull();
    expect(screen.getByPlaceholderText("/value")).toBeTruthy();
    expect(screen.queryByText("Time format")).toBeNull();
  });
});
