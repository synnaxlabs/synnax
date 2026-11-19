// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
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
import { describe, expect, it } from "vitest";

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

  it("should throw from useViews outside a Dialog.Views", () => {
    expect(() => renderHook(() => Dialog.useViews())).toThrow(
      "Dialog.useViews must be used within Dialog.Views",
    );
  });
});
