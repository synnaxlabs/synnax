// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { ToggleCommand } from "@/layered/service/theme/ToggleCommand";

const renderCommand = (toggleTheme: () => void) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Theming.Provider toggleTheme={toggleTheme}>{children}</Theming.Provider>
  );
  return render(
    <ToggleCommand key={ToggleCommand.key} itemKey={ToggleCommand.key} index={0} />,
    { wrapper: Wrapper },
  );
};

describe("theme/ToggleCommand", () => {
  it("should expose the correct command metadata", () => {
    expect(ToggleCommand.key).toBe("toggle-theme");
    expect(ToggleCommand.commandName).toBe("Toggle color theme");
  });

  it("should render the toggle theme item", () => {
    renderCommand(vi.fn());
    expect(screen.getByText("Toggle color theme")).toBeDefined();
  });

  it("should toggle the theme when selected", () => {
    const toggleTheme = vi.fn();
    renderCommand(toggleTheme);
    fireEvent.click(screen.getByText("Toggle color theme"));
    expect(toggleTheme).toHaveBeenCalledTimes(1);
  });
});
