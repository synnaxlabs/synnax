// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { getBySelector, renderWithConsole } from "@/testutil";

describe("Controls.Frame", () => {
  it("should apply the expanded modifier class only while expanded", async () => {
    const { container, rerender } = await renderWithConsole(
      <Task.Controls.Frame>
        <span>a</span>
      </Task.Controls.Frame>,
    );
    expect(container.querySelector(".console-task-controls--expanded")).toBeNull();

    rerender(
      <Task.Controls.Frame expanded>
        <span>a</span>
      </Task.Controls.Frame>,
    );
    expect(container.querySelector(".console-task-controls--expanded")).toBeTruthy();
  });

  it("should invoke onContract when clicked while expanded", async () => {
    const onContract = vi.fn();
    const { container } = await renderWithConsole(
      <Task.Controls.Frame expanded onContract={onContract}>
        <span>c</span>
      </Task.Controls.Frame>,
    );
    fireEvent.click(getBySelector(container, ".console-task-controls"));
    expect(onContract).toHaveBeenCalledTimes(1);
  });

  it("should not invoke onContract when clicked while collapsed", async () => {
    const onContract = vi.fn();
    const { container } = await renderWithConsole(
      <Task.Controls.Frame onContract={onContract}>
        <span>d</span>
      </Task.Controls.Frame>,
    );
    fireEvent.click(getBySelector(container, ".console-task-controls"));
    expect(onContract).not.toHaveBeenCalled();
  });
});
