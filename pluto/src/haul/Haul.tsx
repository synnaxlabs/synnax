// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/haul/Haul.css";

import { type destructor, type optional, type record, xy } from "@synnaxlabs/x";
import React, {
  type DragEvent,
  type DragEventHandler,
  memo,
  type PropsWithChildren,
  type ReactNode,
  type RefObject,
  useCallback,
  useId,
  useMemo,
  useRef,
} from "react";
import { z } from "zod";

import { context } from "@/context";
import { type state } from "@/state";

/** Zod schema for a draggable/droppable {@link Item}. */
export const itemZ = z.object({
  key: z.string().or(z.number()),
  type: z.string(),
  elementID: z.string().optional(),
  data: z.unknown().optional(),
});

/**
 * A draggable/droppable payload identified by `type` and `key`. `data` is
 * required unless `Data` includes `undefined`.
 */
export type Item<
  Type extends string = string,
  Key extends record.Key = record.Key,
  Data = unknown,
> = {
  key: Key;
  type: Type;
  elementID?: string;
} & (undefined extends Data ? { data?: Data } : { data: Data });

/** Zod schema for {@link DraggingState}. */
export const draggingStateZ = z.object({ source: itemZ, items: z.array(itemZ) });

/** The item a drag originated from (`source`) and everything hauled with it. */
export interface DraggingState {
  source: Item;
  items: Item[];
}

/** Empty {@link Item}, used as a zero value. */
export const ZERO_ITEM: Item = { key: "", type: "" };

/** Empty {@link DraggingState}, used as a zero value. */
export const ZERO_DRAGGING_STATE: DraggingState = { source: ZERO_ITEM, items: [] };

/** The drop target and the items that landed on it. */
interface DropProps {
  target: Item;
  dropped: Item[];
}

/** Item type used to represent an OS file drag. */
export const FILE_TYPE = "file";

/** Sentinel {@link Item} representing an OS file drag. */
export const FILE: Item = { key: "file", type: FILE_TYPE };

// Effects that indicate a file is being dragged. Downside is that this also
// allows dragging links, but that's not a huge deal.
const ALLOWED_FILE_DRAG_EFFECTS = new Set(["all", "copyLink"]);

/** Reports whether `event` is an OS file drag rather than an internal haul drag. */
export const isFileDrag = (event: DragEvent, dragging: DraggingState): boolean =>
  ALLOWED_FILE_DRAG_EFFECTS.has(event.dataTransfer.effectAllowed) &&
  dragging.items.length === 0;

/** Called on drag end; returns drop props to finalize, or null to ignore. */
interface DragEndInterceptor {
  (state: DraggingState, cursor: xy.XY): DropProps | null;
}

/**
 * Haul drag-and-drop controls: the imperative drag/drop callbacks plus a live-state
 * ref. Its identity is stable across drags, so subscribing to it does not re-render
 * on drag-state changes. Read the live dragging state via {@link useDraggingState}
 * (re-renders) or {@link useDraggingRef} (does not).
 */
export interface ContextValue {
  /** Begins a drag of `items` originating from `source`. */
  start: (
    source: Item,
    items: Item[],
    onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void,
  ) => void;
  /** Ends the drag at `cursor`, running bound interceptors to resolve any drop. */
  end: (cursor: xy.XY) => void;
  /** Finalizes a drop, invoking the drag's `onSuccessfulDrop` callback. */
  drop: (props: DropProps) => void;
  /** Registers an interceptor `end` consults to resolve drops outside a drop target. */
  bind: (interceptor: DragEndInterceptor) => destructor.Destructor;
  // claimDropEvent atomically claims a native drop event for a single drop target,
  // returning false if another target already claimed it. Drop events bubble, so
  // the innermost target claims first and nested targets do not multi-fire.
  claimDropEvent: (event: Event) => boolean;
  /** A live ref to the current dragging state, readable at event time. */
  stateRef: RefObject<DraggingState>;
}

const [Context, useContext] = context.create<ContextValue | null>({
  defaultValue: null,
  displayName: "Haul.Context",
});
/**
 * Accesses the ambient drag/drop controls; null if no {@link Provider} is mounted.
 * The controls are identity-stable, so consuming them does not re-render on drag-state
 * changes.
 */
export { useContext };

const [StateContext, useStateContext] = context.create<DraggingState>({
  defaultValue: ZERO_DRAGGING_STATE,
  displayName: "Haul.StateContext",
});

/** Props for {@link Provider}. */
export interface ProviderProps extends PropsWithChildren {
  /** State hook backing the drag; defaults to `React.useState`. */
  useState?: state.PureUse<DraggingState>;
}

interface ProviderRef extends DraggingState {
  onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void;
}

const HAUL_REF: ProviderRef = {
  ...ZERO_DRAGGING_STATE,
  onSuccessfulDrop: () => {},
};

/** Provides haul drag-and-drop state to descendants. */
export const Provider = memo(
  ({ children, useState = React.useState }: ProviderProps): ReactNode => {
    const ctx = useContext();

    const [state, setState] = useState(ZERO_DRAGGING_STATE);
    const ref = useRef<ProviderRef>(HAUL_REF);
    const interceptors = useRef<Set<DragEndInterceptor>>(new Set());
    const handledEvents = useRef<WeakSet<Event>>(new WeakSet());

    const claimDropEvent: ContextValue["claimDropEvent"] = useCallback((event) => {
      if (handledEvents.current.has(event)) return false;
      handledEvents.current.add(event);
      return true;
    }, []);

    const start: ContextValue["start"] = useCallback(
      (source, items, onSuccessfulDrop) => {
        ref.current = { source, items, onSuccessfulDrop };
        setState({ source, items });
      },
      [setState],
    );

    const drop: ContextValue["drop"] = useCallback(
      ({ target, dropped }) => {
        const hauled = ref.current.items;
        ref.current.onSuccessfulDrop?.({ target, dropped, hauled });
        ref.current = HAUL_REF;
        setState(ZERO_DRAGGING_STATE);
      },
      [setState],
    );

    const end: ContextValue["end"] = useCallback(
      (cursor: xy.XY) => {
        let dropped: DropProps | null = null;
        interceptors.current.forEach((interceptor) => {
          if (dropped != null) return;
          dropped = interceptor(ref.current, cursor);
        });
        if (dropped != null) drop(dropped);
        ref.current = HAUL_REF;
        setState(ZERO_DRAGGING_STATE);
      },
      [setState],
    );

    const bind: ContextValue["bind"] = useCallback((interceptor) => {
      interceptors.current.add(interceptor);
      return () => interceptors.current.delete(interceptor);
    }, []);

    const controls = useMemo<ContextValue>(
      () => ({ start, end, drop, bind, claimDropEvent, stateRef: ref }),
      [start, end, drop, bind, claimDropEvent],
    );
    // A nested Provider reuses the outer one's contexts, which are already in scope,
    // so it renders children without re-providing.
    if (ctx != null) return children;
    return (
      <Context value={controls}>
        <StateContext value={state}>{children}</StateContext>
      </Context>
    );
  },
);
Provider.displayName = "HaulProvider";

/** Returns a ref tracking the live {@link DraggingState} without re-rendering. */
export const useDraggingRef = (): RefObject<DraggingState> => {
  const fallback = useRef<DraggingState>(ZERO_DRAGGING_STATE);
  const ctx = useContext();
  return ctx?.stateRef ?? fallback;
};

/** Returns the current {@link DraggingState}, re-rendering on change. */
export const useDraggingState = (): DraggingState => useStateContext();

/** Passed to a drag's `onSuccessfulDrop` callback once a drop completes. */
export interface OnSuccessfulDropProps {
  target: Item;
  hauled: Item[];
  dropped: Item[];
}

/** Props for {@link useDrag}; `key` defaults to a generated ID. */
export interface UseDragProps extends optional.Optional<Item, "key"> {}

/** Return value of {@link useDrag}. */
export interface UseDragReturn {
  /** Begins hauling `items`, invoking `onSuccessfulDrop` if they're dropped. */
  startDrag: (
    items: Item[],
    onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void,
  ) => void;
  /** Native `dragend`/`mouseup` handler that resolves the drag. */
  onDragEnd: (e: DragEvent | MouseEvent) => void;
}

/** Wires an element up as a drag source of `type`. */
export const useDrag = ({ type, key }: UseDragProps): UseDragReturn => {
  const generatedKey = useId();
  key ??= generatedKey;
  const source: Item = useMemo(() => ({ key, type }), [key, type]);
  const ctx = useContext();
  if (ctx == null) return { startDrag: () => {}, onDragEnd: () => {} };
  const { start, end } = ctx;
  return {
    startDrag: useCallback((items, f) => start(source, items, f), [start, source]),
    onDragEnd: (e: DragEvent | MouseEvent) =>
      end(xy.construct({ x: e.screenX, y: e.screenY })),
  };
};

/** Predicate deciding whether a drop target accepts the current drag. */
export interface CanDrop {
  (state: DraggingState): boolean;
}

/** The {@link DraggingState} at drop time, plus the originating native event. */
export interface OnDropProps extends DraggingState {
  event?: DragEvent;
}

/** Handles a drop, returning the items actually consumed. */
export interface OnDrop {
  (props: OnDropProps): Item[];
}

/** Props passed to a drop target's `onDragOver` callback. */
export interface OnDragOverProps extends OnDropProps {}

/** Props for {@link useDrop}; `key` defaults to a generated ID. */
export interface UseDropProps extends optional.Optional<Item, "key"> {
  canDrop: CanDrop;
  onDrop: OnDrop;
  onDragOver?: (props: OnDragOverProps) => void;
}

/** Return value of {@link useDrop}: DOM handlers to spread onto a drop target. */
export interface UseDropReturn {
  onDragOver: DragEventHandler;
  onDrop: DragEventHandler;
}

/** Wires an element up as a drop target of `type`. */
export const useDrop = ({
  type,
  key,
  canDrop,
  onDrop,
  onDragOver,
}: UseDropProps): UseDropReturn => {
  const ref = useDraggingRef();
  const ctx = useContext();
  if (ctx == null) return { onDragOver: () => {}, onDrop: () => {} };
  const { drop, claimDropEvent } = ctx;

  const generatedKey = useId();
  key ??= generatedKey;
  const target: Item = useMemo(() => ({ key, type }), [key, type]);

  const prevDragOver = useRef<xy.XY>({ x: -1, y: -1 });

  const handleDragOver = useCallback(
    (event: DragEvent) => {
      event.preventDefault();
      const cursor = xy.construct({ x: event.screenX, y: event.screenY });
      if (xy.equals(cursor, prevDragOver.current)) return;
      prevDragOver.current = cursor;
      if (!canDrop(ref.current)) return;
      onDragOver?.({ event, ...ref.current });
    },
    [ref, canDrop],
  );

  const handleDrop = useCallback(
    (event: DragEvent) => {
      if (!canDrop(ref.current)) return;
      if (!claimDropEvent(event.nativeEvent)) return;
      event.preventDefault();
      drop({ target, dropped: onDrop({ ...ref.current, event }) });
    },
    [ref, onDrop, canDrop, drop, claimDropEvent, target],
  );

  return { onDragOver: handleDragOver, onDrop: handleDrop };
};

/** Props for {@link useDragAndDrop}; `type`/`key` serve as both source and target. */
export interface UseDragAndDropProps
  extends
    Omit<UseDragProps, "source">,
    Omit<UseDropProps, "target">,
    optional.Optional<Item, "key"> {}

/** Return value of {@link useDragAndDrop}. */
export interface UseDragAndDropReturn extends UseDragReturn, UseDropReturn {}

/** Wires an element up as both a drag source and a drop target of `type`. */
export const useDragAndDrop = ({
  type,
  key,
  ...rest
}: UseDragAndDropProps): UseDragAndDropReturn => {
  const generatedKey = useId();
  key ??= generatedKey;
  const sourceAndTarget: Item = useMemo(() => ({ key, type }), [key, type]);
  const dragProps = useDrag(sourceAndTarget);
  const dropProps = useDrop({ ...rest, ...sourceAndTarget });
  return { ...dragProps, ...dropProps };
};

/** {@link CanDrop} matching any drag containing at least one item of `type`. */
export const canDropOfType =
  <I extends Item = Item>(type: I["type"]): CanDrop =>
  ({ items }) =>
    items.some((entity) => entity.type === type);

/** Returns the subset of `entities` whose type is `type`. */
export const filterByType = (type: string, entities: Item[]): Item[] =>
  entities.filter((entity) => entity.type === type);

/** Memoizes `fn`, pre-filtering its `items` to `type` before it runs. */
export const useFilterByTypeCallback = (
  type: string,
  fn: OnDrop,
  deps: unknown[],
): OnDrop =>
  useCallback(
    (props) => fn({ ...props, items: filterByType(type, props.items) }),
    deps,
  );

/** Props for a drop handled outside the normal tree; `onDrop` also gets the cursor. */
export interface UseDropOutsideProps extends Omit<UseDropProps, "onDrop"> {
  onDrop: (props: OnDropProps, cursor: xy.XY) => Item[];
}
