// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  act,
  fireEvent,
  render,
  renderHook,
  type RenderResult,
} from "@testing-library/react";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useState,
} from "react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Triggers } from "@/triggers";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>{children}</Triggers.Provider>
);

interface OpenerProps {
  content?: ReactNode;
}

const Opener = ({ content = <span>view body</span> }: OpenerProps): ReactElement => {
  const { open, viewing } = Dialog.useViews();
  return (
    <button onClick={() => open("Edit item", content)}>
      {viewing ? "open, viewing" : "open"}
    </button>
  );
};

const renderViews = (children: ReactNode): RenderResult =>
  render(<Dialog.Views>{children}</Dialog.Views>, { wrapper: Wrapper });

const boxOf = (c: RenderResult): Element | null =>
  c.getByText(/^open/).closest(`.${CSS.B("views")}`);

describe("Views", () => {
  it("should show a view in place of the children until Back", () => {
    const c = renderViews(
      <>
        <Opener />
        <input aria-label="Name" defaultValue="" />
      </>,
    );
    const viewing = CSS.BM("views", "viewing");
    fireEvent.change(c.getByLabelText("Name"), { target: { value: "pump" } });
    fireEvent.click(c.getByText("open"));
    expect(c.getByText("Edit item")).toBeTruthy();
    expect(c.getByText("view body")).toBeTruthy();
    expect(c.getByText("open, viewing")).toBeTruthy();
    expect(boxOf(c)?.classList.contains(viewing)).toBe(true);
    fireEvent.click(c.getByLabelText("Back"));
    expect(c.queryByText("view body")).toBeNull();
    expect(c.getByText("open")).toBeTruthy();
    expect(boxOf(c)?.classList.contains(viewing)).toBe(false);
    expect((c.getByLabelText("Name") as HTMLInputElement).value).toBe("pump");
  });

  it("should focus Back in a view, then the opener after Back", () => {
    const c = renderViews(<Opener />);
    const open = c.getByText("open");
    open.focus();
    fireEvent.click(open);
    expect(document.activeElement).toBe(c.getByLabelText("Back"));
    fireEvent.click(c.getByLabelText("Back"));
    expect(document.activeElement).toBe(open);
  });

  it("should focus the box after Back when the opener is gone", () => {
    const RemovableOpener = (): ReactElement | null => {
      const [present, setPresent] = useState(true);
      if (!present) return null;
      const remove = <button onClick={() => setPresent(false)}>remove</button>;
      return <Opener content={remove} />;
    };
    const c = renderViews(<RemovableOpener />);
    const box = boxOf(c);
    const open = c.getByText("open");
    open.focus();
    fireEvent.click(open);
    fireEvent.click(c.getByText("remove"));
    fireEvent.click(c.getByLabelText("Back"));
    expect(document.activeElement).toBe(box);
  });

  it("should report each view that opens and each Back", () => {
    const onViewingChange = vi.fn();
    const c = render(
      <Dialog.Views onViewingChange={onViewingChange}>
        <Opener />
      </Dialog.Views>,
      { wrapper: Wrapper },
    );
    expect(onViewingChange).not.toHaveBeenCalled();
    fireEvent.click(c.getByText("open"));
    expect(onViewingChange.mock.calls).toEqual([[true]]);
    fireEvent.click(c.getByLabelText("Back"));
    expect(onViewingChange.mock.calls).toEqual([[true], [false]]);
  });

  it("should run the callback of back once the children show, not the focus return", () => {
    let childrenShown: boolean | undefined;
    const Done = (): ReactElement => {
      const { back } = Dialog.useViews();
      return (
        <button
          onClick={() =>
            back(() => {
              childrenShown = c.queryByText("done") == null;
            })
          }
        >
          done
        </button>
      );
    };
    const c = renderViews(<Opener content={<Done />} />);
    const opener = c.getByText("open");
    fireEvent.click(opener);
    fireEvent.click(c.getByText("done"));
    expect(childrenShown).toBe(true);
    expect(document.activeElement).not.toBe(opener);
  });

  it("should run the callback of back at once from a view that closed", () => {
    let back: ((then?: () => void) => void) | undefined;
    const Late = (): ReactElement => {
      back = Dialog.useViews().back;
      return <span>view body</span>;
    };
    const c = renderViews(<Opener content={<Late />} />);
    fireEvent.click(c.getByText("open"));
    fireEvent.click(c.getByLabelText("Back"));
    const then = vi.fn();
    act(() => back?.(then));
    expect(then).toHaveBeenCalledOnce();
  });

  it("should throw from useViews outside a Dialog.Views", () => {
    expect(() => renderHook(() => Dialog.useViews())).toThrow(
      "Dialog.useViews must be used within Dialog.Views",
    );
  });
});
