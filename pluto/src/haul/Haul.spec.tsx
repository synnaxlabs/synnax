// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createEvent,
  fireEvent,
  render,
  renderHook,
  screen,
} from "@testing-library/react";
import {
  type DragEvent,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
} from "react";
import { describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";

const ITEMS: Haul.Item[] = [{ type: "cat", key: "whiskers" }];

interface SourceProps {
  items?: Haul.Item[];
  onSuccessfulDrop?: (props: Haul.OnSuccessfulDropProps) => void;
}

const Source = ({ items = ITEMS, onSuccessfulDrop }: SourceProps): ReactElement => {
  const { startDrag, onDragEnd } = Haul.useDrag({ type: "test-source", key: "source" });
  return (
    <button
      data-testid="source"
      onClick={() => startDrag(items, onSuccessfulDrop)}
      onDragEnd={onDragEnd}
    />
  );
};

const KeylessSource = ({
  items = ITEMS,
  onSuccessfulDrop,
}: SourceProps): ReactElement => {
  const { startDrag } = Haul.useDrag({ type: "keyless-source" });
  return (
    <button
      data-testid="keyless-source"
      onClick={() => startDrag(items, onSuccessfulDrop)}
    />
  );
};

interface ToggleKeySourceProps {
  keyed: boolean;
}

const ToggleKeySource = ({ keyed }: ToggleKeySourceProps): ReactElement => {
  const { startDrag } = Haul.useDrag({
    type: "toggle-source",
    key: keyed ? "fixed-key" : undefined,
  });
  return <button data-testid="toggle-source" onClick={() => startDrag(ITEMS)} />;
};

type DragEndInterceptor = Parameters<Haul.ContextValue["bind"]>[0];

interface InterceptorProps {
  intercept: DragEndInterceptor;
}

const Interceptor = ({ intercept }: InterceptorProps): null => {
  const ctx = Haul.useContext();
  useEffect(() => {
    if (ctx == null) return;
    return ctx.bind(intercept);
  }, [ctx, intercept]);
  return null;
};

interface DnDItemProps {
  itemKey: string;
  onDrop: Haul.OnDrop;
}

const DnDItem = ({ itemKey, onDrop }: DnDItemProps): ReactElement => {
  const dragAndDrop = Haul.useDragAndDrop({
    type: "reorder-item",
    key: itemKey,
    canDrop: Haul.canDropOfType("reorder-item"),
    onDrop,
  });
  return (
    <div
      data-testid={`item-${itemKey}`}
      onDragOver={dragAndDrop.onDragOver}
      onDrop={dragAndDrop.onDrop}
      onClick={() => dragAndDrop.startDrag([{ type: "reorder-item", key: itemKey }])}
    />
  );
};

const captureDragOverEvent = (effectAllowed: string): DragEvent => {
  let captured: DragEvent | undefined;
  render(
    <div
      data-testid="drag-over-probe"
      onDragOver={(e) => {
        captured = e;
      }}
    />,
  );
  fireEvent.dragOver(screen.getByTestId("drag-over-probe"), {
    dataTransfer: { effectAllowed },
  });
  if (captured == null) throw new Error("drag over event was not captured");
  return captured;
};

interface TargetProps {
  testId: string;
  canDrop?: Haul.CanDrop;
  onDrop: Haul.OnDrop;
  onDragOver?: (props: Haul.OnDragOverProps) => void;
  children?: ReactNode;
}

const Target = ({
  testId,
  canDrop = ({ items }) => items.length > 0,
  onDrop,
  onDragOver,
  children,
}: TargetProps): ReactElement => {
  const dropProps = Haul.useDrop({
    type: "test-target",
    key: testId,
    canDrop,
    onDrop,
    onDragOver,
  });
  return (
    <div data-testid={testId} {...dropProps}>
      {children}
    </div>
  );
};

const beginDrag = (): void => {
  fireEvent.click(screen.getByTestId("source"));
};

// jsdom has no native DragEvent constructor, so screenX/screenY passed through
// fireEvent.dragEnd's MouseEventInit are silently dropped; assign them onto the
// raw event before dispatch instead.
const fireDragEnd = (element: HTMLElement, screenX: number, screenY: number): void => {
  const event = createEvent.dragEnd(element);
  Object.assign(event, { screenX, screenY });
  fireEvent(element, event);
};

const passthroughDrop = () => vi.fn(({ items }: Haul.OnDropProps) => items);

describe("Haul", () => {
  describe("re-render isolation", () => {
    it("should not re-render a useDrag consumer when the dragging state changes", () => {
      let dragRenders = 0;
      const DragConsumer = (): ReactElement => {
        Haul.useDrag({ type: "counter-source", key: "counter" });
        dragRenders++;
        return <span />;
      };
      let stateRenders = 0;
      const StateConsumer = (): ReactElement => {
        Haul.useDraggingState();
        stateRenders++;
        return <span />;
      };
      render(
        <Haul.Provider>
          <Source />
          <DragConsumer />
          <StateConsumer />
        </Haul.Provider>,
      );
      const dragBefore = dragRenders;
      const stateBefore = stateRenders;
      beginDrag();
      expect(dragRenders).toBe(dragBefore);
      expect(stateRenders).toBeGreaterThan(stateBefore);
    });
  });

  describe("useDrop", () => {
    it("should call onDrop with the dragged items on a valid drop", () => {
      const onDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <Source />
          <Target testId="target" onDrop={onDrop} />
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("target"));
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop.mock.calls[0][0].items).toEqual(ITEMS);
      expect(onDrop.mock.calls[0][0].source).toEqual({
        type: "test-source",
        key: "source",
      });
    });

    it("should not call onDrop when canDrop rejects the dragged items", () => {
      const onDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <Source />
          <Target testId="target" canDrop={() => false} onDrop={onDrop} />
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("target"));
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should not call onDrop when nothing is being dragged", () => {
      const onDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <Source />
          <Target testId="target" onDrop={onDrop} />
        </Haul.Provider>,
      );
      fireEvent.drop(screen.getByTestId("target"));
      expect(onDrop).not.toHaveBeenCalled();
    });

    it("should clear the dragging state after a successful drop", () => {
      const onDrop = passthroughDrop();
      const { result } = renderHook(() => Haul.useDraggingState(), {
        wrapper: ({ children }: PropsWithChildren) => (
          <Haul.Provider>
            <Source />
            <Target testId="target" onDrop={onDrop} />
            {children}
          </Haul.Provider>
        ),
      });
      expect(result.current.items).toEqual([]);
      beginDrag();
      expect(result.current.items).toEqual(ITEMS);
      fireEvent.drop(screen.getByTestId("target"));
      expect(result.current.items).toEqual([]);
    });

    it("should notify the drag source of a successful drop", () => {
      const onDrop = passthroughDrop();
      const onSuccessfulDrop = vi.fn();
      render(
        <Haul.Provider>
          <Source onSuccessfulDrop={onSuccessfulDrop} />
          <Target testId="target" onDrop={onDrop} />
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("target"));
      expect(onSuccessfulDrop).toHaveBeenCalledTimes(1);
      const props = onSuccessfulDrop.mock.calls[0][0] as Haul.OnSuccessfulDropProps;
      expect(props.hauled).toEqual(ITEMS);
      expect(props.dropped).toEqual(ITEMS);
      expect(props.target).toEqual({ type: "test-target", key: "target" });
    });
  });

  describe("key generation", () => {
    it("should generate a stable key when key is omitted, reused across drags", () => {
      const onDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <KeylessSource />
          <Target testId="target" onDrop={onDrop} />
        </Haul.Provider>,
      );
      fireEvent.click(screen.getByTestId("keyless-source"));
      fireEvent.drop(screen.getByTestId("target"));
      const firstKey = onDrop.mock.calls[0][0].source.key;
      expect(firstKey).toBeTruthy();

      fireEvent.click(screen.getByTestId("keyless-source"));
      fireEvent.drop(screen.getByTestId("target"));
      const secondKey = onDrop.mock.calls[1][0].source.key;
      expect(secondKey).toEqual(firstKey);
    });

    it("should not break the source's hooks when key toggles between provided and omitted across re-renders", () => {
      const onDrop = passthroughDrop();
      const { rerender } = render(
        <Haul.Provider>
          <ToggleKeySource keyed={false} />
          <Target testId="target" onDrop={onDrop} />
        </Haul.Provider>,
      );
      expect(() => {
        rerender(
          <Haul.Provider>
            <ToggleKeySource keyed />
            <Target testId="target" onDrop={onDrop} />
          </Haul.Provider>,
        );
      }).not.toThrow();
      expect(() => {
        rerender(
          <Haul.Provider>
            <ToggleKeySource keyed={false} />
            <Target testId="target" onDrop={onDrop} />
          </Haul.Provider>,
        );
      }).not.toThrow();
      fireEvent.click(screen.getByTestId("toggle-source"));
      fireEvent.drop(screen.getByTestId("target"));
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop.mock.calls[0][0].items).toEqual(ITEMS);
    });
  });

  describe("nested drop targets", () => {
    it("should fire only the innermost target when both can accept the drop", () => {
      const onOuterDrop = passthroughDrop();
      const onInnerDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <Source />
          <Target testId="outer" onDrop={onOuterDrop}>
            <Target testId="inner" onDrop={onInnerDrop} />
          </Target>
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("inner"));
      expect(onInnerDrop).toHaveBeenCalledTimes(1);
      expect(onOuterDrop).not.toHaveBeenCalled();
    });

    it("should fire the outer target when the inner target cannot accept the drop", () => {
      const onOuterDrop = passthroughDrop();
      const onInnerDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <Source />
          <Target testId="outer" onDrop={onOuterDrop}>
            <Target testId="inner" canDrop={() => false} onDrop={onInnerDrop} />
          </Target>
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("inner"));
      expect(onInnerDrop).not.toHaveBeenCalled();
      expect(onOuterDrop).toHaveBeenCalledTimes(1);
    });

    it("should fire the outer target when the drop lands outside the inner target", () => {
      const onOuterDrop = passthroughDrop();
      const onInnerDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <Source />
          <Target testId="outer" onDrop={onOuterDrop}>
            <Target testId="inner" onDrop={onInnerDrop} />
          </Target>
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("outer"));
      expect(onInnerDrop).not.toHaveBeenCalled();
      expect(onOuterDrop).toHaveBeenCalledTimes(1);
    });

    it("should fire once when one handler is spread across nested elements", () => {
      const onDrop = passthroughDrop();
      const SharedHandlerTarget = (): ReactElement => {
        const dropProps = Haul.useDrop({
          type: "test-target",
          key: "shared",
          canDrop: ({ items }) => items.length > 0,
          onDrop,
        });
        return (
          <div data-testid="shared-outer" {...dropProps}>
            <div data-testid="shared-inner" {...dropProps} />
          </div>
        );
      };
      render(
        <Haul.Provider>
          <Source />
          <SharedHandlerTarget />
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("shared-inner"));
      expect(onDrop).toHaveBeenCalledTimes(1);
    });

    it("should claim per-event so sequential drops each fire exactly once", () => {
      const onOuterDrop = passthroughDrop();
      const onInnerDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <Source />
          <Target testId="outer" onDrop={onOuterDrop}>
            <Target testId="inner" onDrop={onInnerDrop} />
          </Target>
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("inner"));
      beginDrag();
      fireEvent.drop(screen.getByTestId("inner"));
      expect(onInnerDrop).toHaveBeenCalledTimes(2);
      expect(onOuterDrop).not.toHaveBeenCalled();
    });

    it("should fire only the inner target's onDragOver when both targets accept", () => {
      const onInnerDragOver = vi.fn();
      const onOuterDragOver = vi.fn();
      render(
        <Haul.Provider>
          <Source />
          <Target
            testId="outer"
            onDrop={passthroughDrop()}
            onDragOver={onOuterDragOver}
          >
            <Target
              testId="inner"
              onDrop={passthroughDrop()}
              onDragOver={onInnerDragOver}
            />
          </Target>
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.dragOver(screen.getByTestId("inner"));
      expect(onInnerDragOver).toHaveBeenCalledTimes(1);
      expect(onOuterDragOver).not.toHaveBeenCalled();
    });

    it("should fire the outer target's onDragOver when the inner target cannot accept", () => {
      const onInnerDragOver = vi.fn();
      const onOuterDragOver = vi.fn();
      render(
        <Haul.Provider>
          <Source />
          <Target
            testId="outer"
            onDrop={passthroughDrop()}
            onDragOver={onOuterDragOver}
          >
            <Target
              testId="inner"
              canDrop={() => false}
              onDrop={passthroughDrop()}
              onDragOver={onInnerDragOver}
            />
          </Target>
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.dragOver(screen.getByTestId("inner"));
      expect(onInnerDragOver).not.toHaveBeenCalled();
      expect(onOuterDragOver).toHaveBeenCalledTimes(1);
    });
  });

  describe("drag end without a drop", () => {
    it("should clear the dragging state when the drag ends and no interceptor claims it", () => {
      const { result } = renderHook(() => Haul.useDraggingState(), {
        wrapper: ({ children }: PropsWithChildren) => (
          <Haul.Provider>
            <Source />
            {children}
          </Haul.Provider>
        ),
      });
      beginDrag();
      expect(result.current.items).toEqual(ITEMS);
      fireDragEnd(screen.getByTestId("source"), 500, 500);
      expect(result.current.items).toEqual([]);
    });

    it("should resolve the drop through a bound interceptor, passing it the cursor", () => {
      const onSuccessfulDrop = vi.fn();
      const intercept = vi.fn<DragEndInterceptor>((state) =>
        state.items.length > 0
          ? { target: { type: "outside", key: "void" }, dropped: state.items }
          : null,
      );
      render(
        <Haul.Provider>
          <Source onSuccessfulDrop={onSuccessfulDrop} />
          <Interceptor intercept={intercept} />
        </Haul.Provider>,
      );
      beginDrag();
      fireDragEnd(screen.getByTestId("source"), 42, 84);

      expect(intercept).toHaveBeenCalledTimes(1);
      expect(intercept.mock.calls[0][1]).toEqual({ x: 42, y: 84 });

      expect(onSuccessfulDrop).toHaveBeenCalledTimes(1);
      const props = onSuccessfulDrop.mock.calls[0][0] as Haul.OnSuccessfulDropProps;
      expect(props.dropped).toEqual(ITEMS);
      expect(props.target).toEqual({ type: "outside", key: "void" });
    });

    it("should stop consulting an interceptor once it unbinds", () => {
      const onSuccessfulDrop = vi.fn();
      const intercept = vi.fn<DragEndInterceptor>((state) =>
        state.items.length > 0
          ? { target: { type: "outside", key: "void" }, dropped: state.items }
          : null,
      );
      const { rerender } = render(
        <Haul.Provider>
          <Source onSuccessfulDrop={onSuccessfulDrop} />
          <Interceptor intercept={intercept} />
        </Haul.Provider>,
      );
      rerender(
        <Haul.Provider>
          <Source onSuccessfulDrop={onSuccessfulDrop} />
        </Haul.Provider>,
      );
      beginDrag();
      fireDragEnd(screen.getByTestId("source"), 1, 1);
      expect(intercept).not.toHaveBeenCalled();
      expect(onSuccessfulDrop).not.toHaveBeenCalled();
    });
  });

  describe("canDropOfType", () => {
    it("should accept a drag containing at least one item of the given type", () => {
      const canDrop = Haul.canDropOfType("cat");
      expect(
        canDrop({
          source: Haul.ZERO_ITEM,
          items: [
            { type: "dog", key: "1" },
            { type: "cat", key: "2" },
          ],
        }),
      ).toBe(true);
    });

    it("should reject a drag containing no items of the given type", () => {
      const canDrop = Haul.canDropOfType("cat");
      expect(
        canDrop({ source: Haul.ZERO_ITEM, items: [{ type: "dog", key: "1" }] }),
      ).toBe(false);
    });

    it("should reject a drag with no items", () => {
      const canDrop = Haul.canDropOfType("cat");
      expect(canDrop({ source: Haul.ZERO_ITEM, items: [] })).toBe(false);
    });
  });

  describe("filterByType", () => {
    it("should return only the entities matching the given type", () => {
      const entities: Haul.Item[] = [
        { type: "cat", key: "1" },
        { type: "dog", key: "2" },
        { type: "cat", key: "3" },
      ];
      expect(Haul.filterByType("cat", entities)).toEqual([
        { type: "cat", key: "1" },
        { type: "cat", key: "3" },
      ]);
    });

    it("should return an empty array when nothing matches", () => {
      expect(Haul.filterByType("bird", [{ type: "cat", key: "1" }])).toEqual([]);
    });
  });

  describe("useFilterByTypeCallback", () => {
    it("should filter dropped items by type before invoking the handler", () => {
      const captured = vi.fn<(items: Haul.Item[]) => void>();
      const FilterTarget = (): ReactElement => {
        const onDrop = Haul.useFilterByTypeCallback(
          "cat",
          (props) => {
            captured(props.items);
            return props.items;
          },
          [],
        );
        const dropProps = Haul.useDrop({
          type: "test-target",
          key: "filter-target",
          canDrop: () => true,
          onDrop,
        });
        return <div data-testid="filter-target" {...dropProps} />;
      };
      render(
        <Haul.Provider>
          <Source
            items={[
              { type: "cat", key: "1" },
              { type: "dog", key: "2" },
            ]}
          />
          <FilterTarget />
        </Haul.Provider>,
      );
      beginDrag();
      fireEvent.drop(screen.getByTestId("filter-target"));
      expect(captured).toHaveBeenCalledWith([{ type: "cat", key: "1" }]);
    });
  });

  describe("useDragAndDrop", () => {
    it("should let the same element act as both a drag source and a drop target", () => {
      const onDrop = passthroughDrop();
      render(
        <Haul.Provider>
          <DnDItem itemKey="a" onDrop={onDrop} />
          <DnDItem itemKey="b" onDrop={onDrop} />
        </Haul.Provider>,
      );
      fireEvent.click(screen.getByTestId("item-a"));
      fireEvent.drop(screen.getByTestId("item-b"));
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop.mock.calls[0][0].items).toEqual([
        { type: "reorder-item", key: "a" },
      ]);
      expect(onDrop.mock.calls[0][0].source).toEqual({
        type: "reorder-item",
        key: "a",
      });
    });

    it("should not accept a drop of an item of the wrong type", () => {
      const onDrop = passthroughDrop();
      const OtherSource = (): ReactElement => {
        const { startDrag } = Haul.useDrag({ type: "other-item", key: "c" });
        return (
          <button
            data-testid="other-source"
            onClick={() => startDrag([{ type: "other-item", key: "c" }])}
          />
        );
      };
      render(
        <Haul.Provider>
          <OtherSource />
          <DnDItem itemKey="a" onDrop={onDrop} />
        </Haul.Provider>,
      );
      fireEvent.click(screen.getByTestId("other-source"));
      fireEvent.drop(screen.getByTestId("item-a"));
      expect(onDrop).not.toHaveBeenCalled();
    });
  });

  describe("isFileDrag", () => {
    it("should report a file drag when the effect is allowed and nothing is being hauled", () => {
      const event = captureDragOverEvent("copyLink");
      expect(Haul.isFileDrag(event, Haul.ZERO_DRAGGING_STATE)).toBe(true);
    });

    it("should not report a file drag when items are already being hauled", () => {
      const event = captureDragOverEvent("copyLink");
      const dragging: Haul.DraggingState = {
        source: { type: "cat", key: "1" },
        items: [{ type: "cat", key: "1" }],
      };
      expect(Haul.isFileDrag(event, dragging)).toBe(false);
    });

    it("should not report a file drag for a disallowed drag effect", () => {
      const event = captureDragOverEvent("move");
      expect(Haul.isFileDrag(event, Haul.ZERO_DRAGGING_STATE)).toBe(false);
    });
  });
});
