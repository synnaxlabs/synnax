// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { getIconButton, stubClipboardWriteText } from "@/testutil";

describe("layouts.DetailsHeader", () => {
  it("should copy the form value at the given path as JSON", async () => {
    const writeText = stubClipboardWriteText();
    const channels = [{ key: "a", enabled: true }];
    const { container } = await renderInTaskForm(
      <Task.Layouts.DetailsHeader path="config" />,
      { values: { config: { channels } } },
    );
    fireEvent.click(getIconButton(container, "json"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(JSON.parse(writeText.mock.calls[0][0] as string)).toEqual({ channels });
  });

  it("should not copy anything when disabled", async () => {
    const writeText = stubClipboardWriteText();
    const { container } = await renderInTaskForm(
      <Task.Layouts.DetailsHeader path="config" disabled />,
      { values: { config: { channels: [] } } },
    );
    const button = getIconButton(container, "json");
    expect(button.className).toContain("pluto--disabled");
    fireEvent.click(button);
    expect(writeText).not.toHaveBeenCalled();
  });
});
