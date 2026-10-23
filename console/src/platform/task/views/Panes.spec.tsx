// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, screen } from "@testing-library/react";
import { useState } from "react";
import {
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

// The global observer fires once at a fixed size; these specs change the width.
const notifiers = new Set<() => void>();

class WidthResizeObserver {
  private readonly notify: () => void;
  constructor(callback: ResizeObserverCallback) {
    this.notify = () => callback([], this);
  }
  observe(): void {
    notifiers.add(this.notify);
    this.notify();
  }
  unobserve(): void {}
  disconnect(): void {
    notifiers.delete(this.notify);
  }
}

let width = 1000;

const resize = (next: number): void => {
  width = next;
  act(() => notifiers.forEach((notify) => notify()));
};

const Harness = () => {
  const [selected, setSelected] = useState("config.a");
  return (
    <Task.Views.Panes
      listTitle="Channels"
      list={<button onClick={() => setSelected("config.b")}>Select b</button>}
      detailsPath={selected}
    >
      <p>Details body</p>
    </Task.Views.Panes>
  );
};

const renderPanes = async (): Promise<HTMLElement> => {
  const { container } = await renderInTaskForm(<Harness />, {
    values: { config: { a: {}, b: {} } },
  });
  const panes = container.querySelector<HTMLElement>(".console-panes");
  if (panes == null) throw new Error("no panes");
  return panes;
};

const toggle = (panes: HTMLElement): void => {
  const button = panes.querySelector("[aria-expanded]");
  if (button == null) throw new Error("no list toggle");
  fireEvent.click(button);
};

const listOpen = (panes: HTMLElement): boolean =>
  panes.classList.contains("console--list-open");

describe("Task.Views.Panes", () => {
  beforeAll(() => {
    vi.stubGlobal("ResizeObserver", WidthResizeObserver);
  });

  let measure: MockInstance<() => DOMRect>;

  beforeEach(() => {
    width = 1000;
    measure = vi
      .spyOn(Element.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: Element) {
        const w = this.classList.contains("console-panes") ? width : 100;
        return new DOMRect(0, 0, w, 100);
      });
  });

  afterEach(() => {
    measure.mockRestore();
    notifiers.clear();
  });

  it("should hide and show the list in a wide layout", async () => {
    const panes = await renderPanes();
    expect(listOpen(panes)).toBe(true);
    toggle(panes);
    expect(listOpen(panes)).toBe(false);
    toggle(panes);
    expect(listOpen(panes)).toBe(true);
  });

  it("should open the list as a drawer in a narrow layout", async () => {
    const panes = await renderPanes();
    resize(400);
    expect(panes.classList).toContain("console--narrow");
    expect(listOpen(panes)).toBe(false);
    toggle(panes);
    expect(listOpen(panes)).toBe(true);
  });

  it("should close the drawer when the selection changes", async () => {
    const panes = await renderPanes();
    resize(400);
    toggle(panes);
    fireEvent.click(screen.getByText("Select b"));
    expect(listOpen(panes)).toBe(false);
  });

  it("should close the drawer on a click in the details", async () => {
    const panes = await renderPanes();
    resize(400);
    toggle(panes);
    fireEvent.click(screen.getByText("Details body"));
    expect(listOpen(panes)).toBe(false);
  });

  it("should forget an open drawer when the layout widens", async () => {
    const panes = await renderPanes();
    resize(400);
    toggle(panes);
    resize(1000);
    toggle(panes);
    expect(listOpen(panes)).toBe(false);
    resize(400);
    expect(listOpen(panes)).toBe(false);
  });
});
