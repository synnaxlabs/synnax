// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { type ReactElement, useRef } from "react";
import {
  afterAll,
  beforeAll,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";

import { mockBoundingClientRect } from "@/testutil";
import { Triggers } from "@/triggers";

interface DraggableProps {
  onDrag: Triggers.DragCallback;
}

const Draggable = ({ onDrag }: DraggableProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  Triggers.useDrag({ bound: ref, onDrag, triggers: [["MouseLeft"]] });
  return <div ref={ref} data-testid="bound" />;
};

const renderDraggable = (onDrag: Triggers.DragCallback): RenderResult =>
  render(
    <Triggers.Provider>
      <Draggable onDrag={onDrag} />
    </Triggers.Provider>,
  );

const rerenderDraggable = (c: RenderResult, onDrag: Triggers.DragCallback): void =>
  c.rerender(
    <Triggers.Provider>
      <Draggable onDrag={onDrag} />
    </Triggers.Provider>,
  );

const moveListeners = (spy: MockInstance): unknown[] =>
  spy.mock.calls
    .filter(([type]) => type === "mousemove")
    .map(([, listener]) => listener);

describe("useDrag", () => {
  beforeAll(() => {
    vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(
      mockBoundingClientRect(0, 0, 100, 100),
    );
  });
  afterAll(() => vi.restoreAllMocks());

  it("should send moves to the onDrag passed during the drag", () => {
    const first = vi.fn();
    const second = vi.fn();
    const c = renderDraggable(first);
    const bound = c.getByTestId("bound");
    fireEvent.mouseDown(bound, { button: 0, clientX: 10, clientY: 10 });
    rerenderDraggable(c, second);
    fireEvent.mouseMove(window, { clientX: 30, clientY: 30 });
    expect(first).not.toHaveBeenCalledWith(
      expect.objectContaining({ stage: "during" }),
    );
    expect(second).toHaveBeenCalledWith(expect.objectContaining({ stage: "during" }));
  });

  it("should remove its move listener on release after onDrag changes", () => {
    const add = vi.spyOn(window, "addEventListener");
    const remove = vi.spyOn(window, "removeEventListener");
    try {
      const c = renderDraggable(vi.fn());
      const bound = c.getByTestId("bound");
      add.mockClear();
      fireEvent.mouseDown(bound, { button: 0, clientX: 10, clientY: 10 });
      rerenderDraggable(c, vi.fn());
      fireEvent.mouseUp(bound, { button: 0, clientX: 20, clientY: 20 });
      expect(moveListeners(add)).toHaveLength(1);
      expect(moveListeners(remove)).toEqual(moveListeners(add));
    } finally {
      add.mockRestore();
      remove.mockRestore();
    }
  });
});
