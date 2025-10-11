// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { Input } from "@/input";
import { Theming } from "@/theming";

const TestThemeContent = (): ReactElement => {
  const { theme, toggleTheme } = Theming.useContext();
  const [checked, setChecked] = useState(false);
  const handleChange = (value: boolean) => {
    setChecked(value);
    toggleTheme();
  };
  return (
    <div>
      {theme.name}
      <Input.Switch aria-label="theme-switch" value={checked} onChange={handleChange} />
    </div>
  );
};

const TestTheme = (): ReactElement => (
  <Theming.Provider>
    <TestThemeContent />
  </Theming.Provider>
);

describe("Theming", () => {
  it("should render a theme", () => {
    const { getByText } = render(<TestTheme />);
    expect(getByText("Synnax Dark")).toBeTruthy();
  });
  it("should toggle a theme", () => {
    const { getByText, getByLabelText } = render(<TestTheme />);
    expect(getByText("Synnax Dark")).toBeTruthy();
    const btn = getByLabelText("theme-switch");
    fireEvent.click(btn);
    expect(getByText("Synnax Light")).toBeTruthy();
  });
});
