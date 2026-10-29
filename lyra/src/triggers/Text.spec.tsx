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

import { Provider } from "@/triggers/Provider";
import { Text } from "@/triggers/Text";
import { type Trigger } from "@/triggers/triggers";

const TAB: Trigger = ["Control", "T"];

const reserving = (reserved: Trigger[]) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider reserved={reserved}>{children}</Provider>
  );
  return Wrapper;
};

describe("Text", () => {
  it("should render the trigger's keycaps", () => {
    const { container } = render(<Text trigger={TAB} />);
    expect(container.textContent).toContain("T");
  });

  it("should render the arrow keys as arrows", () => {
    const { container } = render(<Text trigger={["ArrowUp", "ArrowDown"]} />);
    expect(container.textContent).toBe("");
    expect(container.querySelector(".pluto-icon--keyboard-arrow-up")).not.toBeNull();
    expect(container.querySelector(".pluto-icon--keyboard-arrow-down")).not.toBeNull();
  });

  it("should render nothing for a reserved trigger", () => {
    const { container } = render(<Text trigger={TAB} />, {
      wrapper: reserving([["T", "Control"]]),
    });
    expect(container.innerHTML).toBe("");
  });

  it("should keep the label of a reserved trigger", () => {
    const { container } = render(<Text trigger={TAB}>New tab</Text>, {
      wrapper: reserving([TAB]),
    });
    expect(container.textContent).toBe("New tab");
  });

  it("should render a trigger the provider does not reserve", () => {
    const { container } = render(<Text trigger={["Control", "S"]} />, {
      wrapper: reserving([TAB]),
    });
    expect(container.textContent).toContain("S");
  });

  it("should put the label before the keys", () => {
    const { container } = render(<Text trigger={["Control", "S"]}>Save</Text>);
    expect(container.textContent).toBe("SaveS");
  });

  it("should start a word key with a capital", () => {
    const { container } = render(<Text trigger={["Escape"]} />);
    expect(container.textContent).toBe("Esc");
  });

  describe("modifier order", () => {
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

    const renderOn = async (userAgent: string, trigger: Trigger): Promise<string[]> => {
      vi.resetModules();
      vi.spyOn(window.navigator, "userAgent", "get").mockReturnValue(userAgent);
      const { Text: PlatformText } = await import("@/triggers/Text");
      const { container } = render(<PlatformText trigger={trigger} />);
      return Array.from(container.querySelectorAll(".pluto-text--keyboard"), capName);
    };

    const MIXED: Trigger = ["MouseLeft", "Control", "K", "Shift", "Alt"];

    it("should follow Apple's order on macOS", async () => {
      expect(await renderOn("Macintosh", MIXED)).toEqual([
        "option",
        "shift",
        "command",
        "K",
        "click",
      ]);
    });

    it("should keep the declared order on Windows, with the mouse last", async () => {
      expect(await renderOn("Windows", MIXED)).toEqual([
        "control",
        "K",
        "shift",
        "Alt",
        "click",
      ]);
    });
  });
});
