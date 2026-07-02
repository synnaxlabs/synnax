// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { renderWithConsole } from "@/testutil";

describe("Controls.ConfigureButton", () => {
  it("should render a Configure button", async () => {
    await renderWithConsole(<Task.Controls.ConfigureButton onClick={vi.fn()} />);
    expect(screen.getByRole("button", { name: /Configure/ })).toBeTruthy();
  });

  it("should invoke onClick when pressed", async () => {
    const onClick = vi.fn();
    await renderWithConsole(<Task.Controls.ConfigureButton onClick={onClick} />);
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should be disabled when the disabled prop is set", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Task.Controls.ConfigureButton onClick={onClick} disabled />,
    );
    const button = screen.getByRole("button", { name: /Configure/ });
    fireEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });
});
