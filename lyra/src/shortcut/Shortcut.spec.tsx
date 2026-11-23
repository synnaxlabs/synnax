// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { runtime } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook } from "@testing-library/react";
import type { PropsWithChildren, ReactElement, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { Shortcut } from "@/shortcut";

interface ProvidesProps extends PropsWithChildren {
  command: string;
  handler: () => void;
}

const Provides = ({ command, handler, children }: ProvidesProps): ReactNode => {
  Shortcut.useCommand(command, handler);
  return children;
};

const press = (init: KeyboardEventInit, target: Element | Window = window): boolean =>
  fireEvent.keyDown(target, init);

const bind = (bindings: Record<string, string[]>, os: runtime.OS = "Windows"): Mock => {
  const handler = vi.fn();
  render(
    <Shortcut.Provider bindings={bindings} os={os}>
      {Object.keys(bindings).map((command) => (
        <Provides key={command} command={command} handler={() => handler(command)} />
      ))}
    </Shortcut.Provider>,
  );
  return handler;
};

describe("Shortcut", () => {
  describe("parse", () => {
    it("should parse the modifiers and the key of a chord", () => {
      expect(Shortcut.parse("Mod+Shift+p")).toEqual([
        { key: "P", mod: true, control: false, alt: false, shift: true },
      ]);
    });

    it("should parse a sequence of chords", () => {
      expect(Shortcut.parse("G G").map((c) => c.key)).toEqual(["G", "G"]);
    });

    it("should format modifiers in a fixed order", () => {
      expect(Shortcut.format(Shortcut.parse("Shift+Alt+Control+Mod+F3"))).toBe(
        "Mod+Control+Alt+Shift+F3",
      );
    });

    it.each([
      ["Ctrl+P", 'Shortcut "Ctrl+P" has an unknown modifier "Ctrl"'],
      ["Mod+Mod+P", 'Shortcut "Mod+Mod+P" repeats "Mod"'],
      ["Mod+Return", 'Shortcut "Mod+Return" has an unknown key "Return"'],
      ["Mod+", 'Shortcut "Mod+" has an unknown key ""'],
    ])("should reject %s", (binding, message) => {
      expect(() => Shortcut.parse(binding)).toThrow(message);
    });
  });

  describe("matching", () => {
    it("should run the command bound to a chord", () => {
      const handler = bind({ "palette.open": ["Mod+P"] });
      press({ key: "p", code: "KeyP", ctrlKey: true });
      expect(handler).toHaveBeenCalledExactlyOnceWith("palette.open");
    });

    it("should prevent the default of a press that runs a command", () => {
      bind({ "palette.open": ["Mod+P"] });
      expect(press({ key: "p", code: "KeyP", ctrlKey: true })).toBe(false);
      expect(press({ key: "o", code: "KeyO", ctrlKey: true })).toBe(true);
    });

    it("should require the modifiers to match exactly", () => {
      const handler = bind({ "palette.open": ["Mod+P"] });
      press({ key: "P", code: "KeyP", ctrlKey: true, shiftKey: true });
      press({ key: "p", code: "KeyP" });
      expect(handler).not.toHaveBeenCalled();
    });

    it("should read Mod as Command and Control as itself on macOS", () => {
      const handler = bind(
        { "palette.open": ["Mod+P"], "line.start": ["Control+A"] },
        "macOS",
      );
      press({ key: "p", code: "KeyP", ctrlKey: true });
      expect(handler).not.toHaveBeenCalled();
      press({ key: "p", code: "KeyP", metaKey: true });
      press({ key: "a", code: "KeyA", ctrlKey: true });
      expect(handler.mock.calls).toEqual([["palette.open"], ["line.start"]]);
    });

    it("should ignore the Windows key outside macOS", () => {
      const handler = bind({ "palette.open": ["Mod+P"] });
      press({ key: "p", code: "KeyP", ctrlKey: true, metaKey: true });
      expect(handler).not.toHaveBeenCalled();
    });

    it("should follow the layout's letter, not its position", () => {
      const handler = bind({ undo: ["Mod+Z"] });
      // AZERTY puts W where QWERTY has Z.
      press({ key: "w", code: "KeyZ", ctrlKey: true });
      expect(handler).not.toHaveBeenCalled();
      press({ key: "z", code: "KeyW", ctrlKey: true });
      expect(handler).toHaveBeenCalledOnce();
    });

    it("should use the position when the character is not ASCII", () => {
      const handler = bind({ undo: ["Mod+Z"], select: ["Alt+A"] }, "macOS");
      press({ key: "я", code: "KeyZ", metaKey: true });
      press({ key: "å", code: "KeyA", altKey: true });
      expect(handler.mock.calls).toEqual([["undo"], ["select"]]);
    });

    it("should use the position on the digit row", () => {
      const handler = bind({ "tab.first": ["Mod+1"] });
      // AZERTY types & on the 1 key.
      press({ key: "&", code: "Digit1", ctrlKey: true });
      expect(handler).toHaveBeenCalledOnce();
    });

    it("should match a symbol that Shift typed with or without Shift", () => {
      const handler = bind({ help: ["?"], sheet: ["Mod+Shift+/"] });
      press({ key: "?", code: "Slash", shiftKey: true });
      press({ key: "?", code: "Slash", ctrlKey: true, shiftKey: true });
      expect(handler.mock.calls).toEqual([["help"], ["sheet"]]);
    });

    it("should not drop a Shift that typed nothing", () => {
      const handler = bind({ sheet: ["Mod+/"] }, "macOS");
      // macOS sends the unshifted character while Command is held.
      press({ key: "/", code: "Slash", metaKey: true, shiftKey: true });
      expect(handler).not.toHaveBeenCalled();
    });

    it("should skip key repeat and input method composition", () => {
      const handler = bind({ "palette.open": ["Mod+P"] });
      press({ key: "p", code: "KeyP", ctrlKey: true, repeat: true });
      press({ key: "p", code: "KeyP", ctrlKey: true, isComposing: true });
      // Safari marks the first key of a composition only with the keyCode.
      // eslint-disable-next-line @typescript-eslint/no-deprecated
      press({ key: "p", code: "KeyP", ctrlKey: true, keyCode: 229 });
      expect(handler).not.toHaveBeenCalled();
    });

    it("should skip a character typed with AltGr", () => {
      const handler = bind({ select: ["Mod+Alt+Q"] });
      press({
        key: "@",
        code: "KeyQ",
        ctrlKey: true,
        altKey: true,
        modifierAltGraph: true,
      });
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe("text fields", () => {
    let input: HTMLInputElement;
    beforeEach(() => {
      input = document.body.appendChild(document.createElement("input"));
    });
    afterEach(() => input.remove());

    it("should keep single keys and Shift chords in the field", () => {
      const handler = bind({ "toolbar.channels": ["C"], next: ["Shift+N"] });
      press({ key: "c", code: "KeyC" }, input);
      press({ key: "N", code: "KeyN", shiftKey: true }, input);
      expect(handler).not.toHaveBeenCalled();
    });

    it("should keep the field's own Mod chords in the field", () => {
      const handler = bind({ copy: ["Mod+C"], redo: ["Mod+Shift+Z"] });
      press({ key: "c", code: "KeyC", ctrlKey: true }, input);
      press({ key: "Z", code: "KeyZ", ctrlKey: true, shiftKey: true }, input);
      expect(handler).not.toHaveBeenCalled();
    });

    it("should pass other Mod chords and Escape to commands", () => {
      const handler = bind({ "palette.open": ["Mod+P"], close: ["Escape"] });
      press({ key: "p", code: "KeyP", ctrlKey: true }, input);
      press({ key: "Escape", code: "Escape" }, input);
      expect(handler.mock.calls).toEqual([["palette.open"], ["close"]]);
    });
  });

  describe("sequences", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });
    afterEach(() => {
      vi.useRealTimers();
    });

    it("should run a sequence on its last chord and show the pending chords", () => {
      const handler = vi.fn();
      const { result } = renderHook(
        () => {
          Shortcut.useCommand("go.home", handler);
          return Shortcut.usePending();
        },
        {
          wrapper: ({ children }) => (
            <Shortcut.Provider bindings={{ "go.home": ["G H"] }} os="Windows">
              {children}
            </Shortcut.Provider>
          ),
        },
      );
      expect(press({ key: "g", code: "KeyG" })).toBe(false);
      expect(Shortcut.format(result.current)).toBe("G");
      expect(handler).not.toHaveBeenCalled();
      press({ key: "h", code: "KeyH" });
      expect(handler).toHaveBeenCalledOnce();
      expect(result.current).toEqual([]);
    });

    it("should run the longest binding a press completes", () => {
      const handler = bind({ "toolbar.open": ["C"], "toolbar.pin": ["C C"] });
      press({ key: "c", code: "KeyC" });
      press({ key: "c", code: "KeyC" });
      expect(handler.mock.calls).toEqual([["toolbar.open"], ["toolbar.pin"]]);
    });

    it("should start over once the sequence times out", () => {
      const handler = bind({ "go.home": ["G H"] });
      press({ key: "g", code: "KeyG" });
      act(() => {
        vi.advanceTimersByTime(Shortcut.SEQUENCE_TIMEOUT.milliseconds);
      });
      press({ key: "h", code: "KeyH" });
      expect(handler).not.toHaveBeenCalled();
    });

    it("should start over when the window loses focus", () => {
      const handler = bind({ "go.home": ["G H"] });
      press({ key: "g", code: "KeyG" });
      act(() => {
        window.dispatchEvent(new Event("blur"));
      });
      press({ key: "h", code: "KeyH" });
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe("areas", () => {
    const renderAreas = (
      ui: ReactElement,
      bindings: Record<string, string[]> = { close: ["Escape"] },
    ): void => {
      render(
        <Shortcut.Provider bindings={bindings} os="Windows">
          {ui}
        </Shortcut.Provider>,
      );
    };
    const escape = (): boolean => press({ key: "Escape", code: "Escape" });

    it("should run the command of the deepest active area", () => {
      const outer = vi.fn();
      const inner = vi.fn();
      renderAreas(
        <Provides command="close" handler={outer}>
          <Shortcut.Area active>
            <Provides command="close" handler={inner} />
          </Shortcut.Area>
        </Provides>,
      );
      escape();
      expect(inner).toHaveBeenCalledOnce();
      expect(outer).not.toHaveBeenCalled();
    });

    it("should pass over an area inside an inactive one", () => {
      const outer = vi.fn();
      const inner = vi.fn();
      renderAreas(
        <Provides command="close" handler={outer}>
          <Shortcut.Area active={false}>
            <Shortcut.Area active>
              <Provides command="close" handler={inner} />
            </Shortcut.Area>
          </Shortcut.Area>
        </Provides>,
      );
      escape();
      expect(outer).toHaveBeenCalledOnce();
      expect(inner).not.toHaveBeenCalled();
    });

    it("should run the area mounted last at equal depth", () => {
      const first = vi.fn();
      const last = vi.fn();
      renderAreas(
        <>
          <Shortcut.Area active>
            <Provides command="close" handler={first} />
          </Shortcut.Area>
          <Shortcut.Area active>
            <Provides command="close" handler={last} />
          </Shortcut.Area>
        </>,
      );
      escape();
      expect(last).toHaveBeenCalledOnce();
      expect(first).not.toHaveBeenCalled();
    });

    it("should pick the deepest area across commands that share a key", () => {
      const exitFocus = vi.fn();
      const closeModal = vi.fn();
      renderAreas(
        <Provides command="focus.exit" handler={exitFocus}>
          <Shortcut.Area active>
            <Provides command="modal.close" handler={closeModal} />
          </Shortcut.Area>
        </Provides>,
        { "focus.exit": ["Escape"], "modal.close": ["Escape"] },
      );
      escape();
      expect(closeModal).toHaveBeenCalledOnce();
      expect(exitFocus).not.toHaveBeenCalled();
    });

    it("should leave a press that no active area provides to the page", () => {
      renderAreas(
        <Shortcut.Area active={false}>
          <Provides command="close" handler={vi.fn()} />
        </Shortcut.Area>,
      );
      expect(escape()).toBe(true);
    });
  });

  describe("useContext", () => {
    const setup = (active: boolean) => {
      const handler = vi.fn();
      const { result } = renderHook(Shortcut.useContext, {
        wrapper: ({ children }) => (
          <Shortcut.Provider
            bindings={{ "palette.open": ["Mod+P", "F1"] }}
            os="Windows"
          >
            <Shortcut.Area active={active}>
              <Provides command="palette.open" handler={handler} />
            </Shortcut.Area>
            {children}
          </Shortcut.Provider>
        ),
      });
      return { handler, context: result.current };
    };

    it("should run a command that an active area provides", () => {
      const { handler, context } = setup(true);
      expect(context.available("palette.open")).toBe(true);
      expect(context.run("palette.open")).toBe(true);
      expect(handler).toHaveBeenCalledOnce();
    });

    it("should not run a command that no active area provides", () => {
      const { handler, context } = setup(false);
      expect(context.available("palette.open")).toBe(false);
      expect(context.run("palette.open")).toBe(false);
      expect(context.run("palette.close")).toBe(false);
      expect(handler).not.toHaveBeenCalled();
    });

    it("should return the bindings of a command", () => {
      const { context } = setup(true);
      expect(context.bindings("palette.open").map(Shortcut.format)).toEqual([
        "Mod+P",
        "F1",
      ]);
      expect(context.bindings("palette.close")).toEqual([]);
    });
  });

  it("should throw when a command is provided outside a Provider", () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => Shortcut.useCommand("close", vi.fn()))).toThrow(
      "Shortcut.useCommand must be used within Shortcut.Provider",
    );
  });
});
