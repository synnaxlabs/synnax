// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { lazy, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Modals } from "@/platform/modals";
import { renderWithModals, Wrapper } from "@/platform/modals/testutil";
import { Session } from "@/session";

interface OpenerProps {
  text: string;
}

const Content: Modals.Content<OpenerProps, void> = ({ text, close }) => (
  <div>
    <span>{text}</span>
    <Button.Button onClick={() => close()}>close-{text}</Button.Button>
  </div>
);

const useOpen = Modals.create(Content);

const Opener = ({ text }: OpenerProps): ReactElement => {
  const open = useOpen();
  return <Button.Button onClick={() => open({ text })}>open-{text}</Button.Button>;
};

const clickText = (text: string): void => {
  const el = screen.getByText(text).closest("button") ?? screen.getByText(text);
  fireEvent.click(el);
};

describe("Stack", () => {
  it("should render nothing when no modals are open", () => {
    renderWithModals(<Opener text="A" />);
    expect(screen.queryByText("A")).toBeNull();
  });

  it("should mount a modal for each open entry", async () => {
    renderWithModals(
      <>
        <Opener text="A" />
        <Opener text="B" />
      </>,
    );
    clickText("open-A");
    clickText("open-B");
    await waitFor(() => {
      expect(screen.getByText("A")).toBeTruthy();
      expect(screen.getByText("B")).toBeTruthy();
    });
  });

  it("should remove only the dismissed modal from the DOM", async () => {
    renderWithModals(
      <>
        <Opener text="A" />
        <Opener text="B" />
      </>,
    );
    clickText("open-A");
    clickText("open-B");
    await waitFor(() => expect(screen.getByText("A")).toBeTruthy());
    clickText("close-A");
    await waitFor(() => expect(screen.queryByText("A")).toBeNull());
    expect(screen.getByText("B")).toBeTruthy();
  });

  it("should dismiss a modal when its dialog requests close", async () => {
    const DialogCloser: Modals.Content<Record<never, never>, void> = () => {
      const { close } = Dialog.useContext();
      return <Button.Button onClick={() => close()}>dialog-close</Button.Button>;
    };
    const useOpenCloser = Modals.create(DialogCloser);
    const CloserOpener = (): ReactElement => {
      const open = useOpenCloser();
      return <Button.Button onClick={() => open()}>open-closer</Button.Button>;
    };
    renderWithModals(<CloserOpener />);
    clickText("open-closer");
    await waitFor(() => expect(screen.getByText("dialog-close")).toBeTruthy());
    clickText("dialog-close");
    await waitFor(() => expect(screen.queryByText("dialog-close")).toBeNull());
  });

  it("should suspense-render lazily loaded modal content", async () => {
    const Lazy = lazy(async () => ({
      default: (): ReactElement => <span>lazy loaded</span>,
    }));
    const LazyContent: Modals.Content<Record<never, never>, void> = () => <Lazy />;
    const useOpenLazy = Modals.create(LazyContent);
    const LazyOpener = (): ReactElement => {
      const open = useOpenLazy();
      return <Button.Button onClick={() => open()}>open-lazy</Button.Button>;
    };
    renderWithModals(<LazyOpener />);
    clickText("open-lazy");
    await waitFor(() => expect(screen.getByText("lazy loaded")).toBeTruthy());
  });

  describe("triggers", () => {
    const Noop: Modals.Content = () => null;

    interface ListenerProps {
      onTrigger?: () => void;
      // Closes its modal on the trigger, ahead of every listener below it.
      selfClosing?: boolean;
    }

    const Listener: Modals.Content<ListenerProps, void> = ({
      onTrigger,
      selfClosing = false,
      close,
    }) => {
      Triggers.use({
        triggers: [["Control", "Enter"]],
        loose: true,
        priority: selfClosing ? 1 : 0,
        callback: ({ stage }) => {
          if (stage !== "start") return;
          onTrigger?.();
          if (selfClosing) close();
        },
      });
      return null;
    };

    const press = (...codes: string[]): void => {
      codes.forEach((code) => fireEvent.keyDown(window, { code }));
      [...codes].reverse().forEach((code) => fireEvent.keyUp(window, { code }));
    };

    /** Renders a page trigger inside the stack, as the window renders its page. */
    const renderPage = (trigger: Triggers.Trigger) => {
      const callback = vi.fn();
      const StackWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <Triggers.Provider>
            <Modals.Stack>{children}</Modals.Stack>
          </Triggers.Provider>
        </Wrapper>
      );
      const { result } = renderHook(
        () => {
          Triggers.use({
            triggers: [trigger],
            loose: true,
            callback: ({ stage }) => {
              if (stage === "start") callback();
            },
          });
          return Session.Modals.useStore("Stack.spec");
        },
        { wrapper: StackWrapper },
      );
      return { callback, modals: result.current };
    };

    it("should withhold page triggers while a modal is open", () => {
      const { callback, modals } = renderPage(["Control", "T"]);
      act(() => modals.push(Noop, {}, () => {}));
      press("ControlLeft", "KeyT");
      expect(callback).not.toHaveBeenCalled();
      act(() => modals.closeTop());
      press("ControlLeft", "KeyT");
      expect(callback).toHaveBeenCalledOnce();
    });

    it("should let only the top modal hear its triggers", () => {
      const { modals } = renderPage(["Control", "T"]);
      const below = vi.fn();
      const above = vi.fn();
      act(() => {
        modals.push(Listener, { onTrigger: below }, () => {});
        modals.push(Listener, { onTrigger: above }, () => {});
      });
      press("ControlLeft", "Enter");
      expect(above).toHaveBeenCalledOnce();
      expect(below).not.toHaveBeenCalled();
      act(() => modals.closeTop());
      press("ControlLeft", "Enter");
      expect(below).toHaveBeenCalledOnce();
    });

    it("should not pass a press below when the top modal closes on it", () => {
      const { modals } = renderPage(["Control", "T"]);
      const below = vi.fn();
      act(() => {
        modals.push(Listener, { onTrigger: below }, () => {});
        modals.push(Listener, { selfClosing: true }, () => {});
      });
      act(() => press("ControlLeft", "Enter"));
      expect(modals.getState()).toHaveLength(1);
      expect(below).not.toHaveBeenCalled();
    });

    it("should not pass a press to the page when the last modal closes on it", () => {
      const { callback, modals } = renderPage(["Control", "Enter"]);
      act(() => modals.push(Listener, { selfClosing: true }, () => {}));
      act(() => press("ControlLeft", "Enter"));
      expect(modals.isAnyOpen()).toBe(false);
      expect(callback).not.toHaveBeenCalled();
    });

    it("should close the top modal on the close trigger, out of the page's hearing", () => {
      const { callback, modals } = renderPage(["Control", "W"]);
      act(() => {
        modals.push(Noop, {}, () => {});
        modals.push(Noop, {}, () => {});
      });
      act(() => press("ControlLeft", "KeyW"));
      expect(modals.getState()).toHaveLength(1);
      act(() => press("ControlLeft", "KeyW"));
      expect(modals.isAnyOpen()).toBe(false);
      expect(callback).not.toHaveBeenCalled();
      press("ControlLeft", "KeyW");
      expect(callback).toHaveBeenCalledOnce();
    });
  });
});
