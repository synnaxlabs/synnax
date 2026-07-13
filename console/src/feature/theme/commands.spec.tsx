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

import { Theme } from "@/feature/theme";

describe("Theme Commands", () => {
  it("should toggle the color theme when the command is selected", () => {
    const toggleTheme = vi.fn();
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Theming.Provider toggleTheme={toggleTheme}>{children}</Theming.Provider>
    );
    const [Toggle] = Theme.COMMANDS;
    render(<Toggle key={Toggle.key} itemKey={Toggle.key} index={0} />, {
      wrapper: Wrapper,
    });
    fireEvent.click(screen.getByText("Toggle color theme"));
    expect(toggleTheme).toHaveBeenCalledTimes(1);
  });
});
