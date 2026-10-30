// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Triggers } from "@/triggers";

const TAB: Triggers.Trigger = ["Control", "T"];

const reserving = (reserved: Triggers.Trigger[]) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Triggers.Provider reserved={reserved}>{children}</Triggers.Provider>
  );
  return Wrapper;
};

describe("Text", () => {
  it("should render the trigger's keycaps", () => {
    const { container } = render(<Triggers.Text trigger={TAB} />);
    expect(container.textContent).toContain("T");
  });

  it("should render the arrow keys as arrows", () => {
    const { container } = render(<Triggers.Text trigger={["ArrowUp", "ArrowDown"]} />);
    expect(container.textContent).toBe("");
    expect(container.querySelector(".lyra-icon--keyboard-arrow-up")).not.toBeNull();
    expect(container.querySelector(".lyra-icon--keyboard-arrow-down")).not.toBeNull();
  });

  it("should render nothing for a reserved trigger", () => {
    const { container } = render(<Triggers.Text trigger={TAB} />, {
      wrapper: reserving([["T", "Control"]]),
    });
    expect(container.innerHTML).toBe("");
  });

  it("should keep the label of a reserved trigger", () => {
    const { container } = render(<Triggers.Text trigger={TAB}>New tab</Triggers.Text>, {
      wrapper: reserving([TAB]),
    });
    expect(container.textContent).toBe("New tab");
  });

  it("should render a trigger the provider does not reserve", () => {
    const { container } = render(<Triggers.Text trigger={["Control", "S"]} />, {
      wrapper: reserving([TAB]),
    });
    expect(container.textContent).toContain("S");
  });

  it("should put the label before the keys", () => {
    const { container } = render(
      <Triggers.Text trigger={["Control", "S"]} os="macOS">
        Save
      </Triggers.Text>,
    );
    expect(container.textContent).toBe("SaveS");
  });

  it("should start a word key with a capital", () => {
    const { container } = render(<Triggers.Text trigger={["Escape"]} />);
    expect(container.textContent).toBe("Esc");
  });

  describe("platform", () => {
    afterEach(() => {
      vi.restoreAllMocks();
      vi.resetModules();
    });

    const capName = (cap: Element): string =>
      cap
        .querySelector("svg")
        ?.getAttribute("class")
        ?.match(/--(?:keyboard-)?(\w+)/)?.[1] ??
      cap.textContent ??
      "";

    const capsOf = (container: HTMLElement): string[] =>
      Array.from(container.querySelectorAll(".lyra-text--keyboard"), capName);

    const MIXED: Triggers.Trigger = ["MouseLeft", "Control", "K", "Shift", "Alt"];

    it("should follow Apple's order and symbols on macOS", () => {
      const { container } = render(<Triggers.Text trigger={MIXED} os="macOS" />);
      expect(capsOf(container)).toEqual(["option", "shift", "command", "K", "click"]);
    });

    it("should keep the declared order on Windows, with the mouse last", () => {
      const { container } = render(<Triggers.Text trigger={MIXED} os="Windows" />);
      expect(capsOf(container)).toEqual(["Ctrl", "K", "Shift", "Alt", "click"]);
    });

    it("should print Linux keys the way Windows prints them", () => {
      const { container } = render(<Triggers.Text trigger={MIXED} os="Linux" />);
      expect(capsOf(container)).toEqual(["Ctrl", "K", "Shift", "Alt", "click"]);
    });

    it("should name the backspace key the way each platform labels it", () => {
      const mac = render(<Triggers.Text trigger={["Backspace"]} os="macOS" />);
      expect(mac.container.textContent).toBe("Delete");
      const windows = render(<Triggers.Text trigger={["Backspace"]} os="Windows" />);
      expect(windows.container.textContent).toBe("Backspace");
    });

    it("should print the keys of the platform the page runs on by default", async () => {
      vi.resetModules();
      vi.spyOn(window.navigator, "userAgent", "get").mockReturnValue("Macintosh");
      const reloaded = await import("@/triggers");
      const { container } = render(<reloaded.Triggers.Text trigger={MIXED} />);
      expect(capsOf(container)).toEqual(["option", "shift", "command", "K", "click"]);
    });
  });
});
