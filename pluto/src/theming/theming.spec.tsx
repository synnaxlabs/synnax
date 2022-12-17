import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Theming } from ".";

const TestThemeContent = (): JSX.Element => {
  const { theme } = Theming.useContext();
  return (
    <div>
      {theme.name}
      <Theming.Switch aria-label="theme-switch" />
    </div>
  );
};

const TestTheme = (): JSX.Element => {
  const a = Theming.useProvider({
    themes: Theming.themes,
    defaultTheme: "synnaxDark",
  });
  return (
    <Theming.Provider {...a}>
      <TestThemeContent />
    </Theming.Provider>
  );
};

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
