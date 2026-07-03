// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { renderWithConsole } from "@/testutil";

const pressConfigureTrigger = () => {
  fireEvent.keyDown(document.body, { code: "ControlLeft" });
  fireEvent.keyDown(document.body, { code: "Enter" });
  fireEvent.keyUp(document.body, { code: "Enter" });
  fireEvent.keyUp(document.body, { code: "ControlLeft" });
};

describe("Controls.ConfigureButton", () => {
  it("should invoke onClick when pressed", async () => {
    const onClick = vi.fn();
    await renderWithConsole(<Task.Controls.ConfigureButton onClick={onClick} />);
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should not invoke onClick when disabled", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Task.Controls.ConfigureButton onClick={onClick} disabled />,
    );
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    expect(onClick).not.toHaveBeenCalled();
  });

  it("should invoke onClick on Control+Enter when showTrigger is set", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Triggers.Provider>
        <Task.Controls.ConfigureButton onClick={onClick} showTrigger />
      </Triggers.Provider>,
    );
    pressConfigureTrigger();
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should ignore Control+Enter when showTrigger is not set", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Triggers.Provider>
        <Task.Controls.ConfigureButton onClick={onClick} />
      </Triggers.Provider>,
    );
    pressConfigureTrigger();
    expect(onClick).not.toHaveBeenCalled();
  });
});
