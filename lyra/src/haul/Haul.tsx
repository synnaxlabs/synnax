// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/haul/Haul.css";

import { type destructor, id, type optional, type record, xy } from "@synnaxlabs/x";
import {
  type DragEvent,
  type DragEventHandler,
  memo,
  type PropsWithChildren,
  type ReactNode,
  type RefObject,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";
import { createPortal, flushSync } from "react-dom";
import { z } from "zod";

import { context } from "@/context";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks/ref";
import { state } from "@/state";

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

/**
 * The drag state that the providers of different windows share. `drop` is absent while
 * the drag is in flight. Once the drag ends, it holds the drop that took the drag, or
 * null when nothing took it.
 */
export interface State extends DraggingState {
  /** Identifies the drag, so a window can match an ended drag to the one it saw. */
  key: string;
  drop?: OnSuccessfulDropProps | null;
}

/** Empty {@link State}, used as a zero value. */
export const ZERO_STATE: State = { ...ZERO_DRAGGING_STATE, key: "" };

/** The drop target and the items that landed on it. */
export interface DropProps {
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
 * Called once per drag when it resolves. `landed` reports whether the items reached a
 * drop target, whether an ordinary one or an interceptor's.
 */
export interface OnResolve {
  (landed: boolean): void;
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
  /**
   * Ends the drag at `cursor`, running bound interceptors to resolve any drop. Set
   * `dropped` when the browser reports a drop: the window it happened in ends the drag.
   */
  end: (cursor: xy.XY, dropped?: boolean) => void;
  /** Finalizes a drop, invoking the drag's `onSuccessfulDrop` callback. */
  drop: (props: DropProps) => void;
  /** Registers an interceptor `end` consults to resolve drops outside a drop target. */
  bind: (interceptor: DragEndInterceptor) => destructor.Destructor;
  /**
   * Registers a watcher run once per drag when it resolves. A watcher cannot claim a
   * drop, so every watcher runs whatever an interceptor did and in any order.
   */
  watch: (onResolve: OnResolve) => destructor.Destructor;
  // claimDragEvent atomically claims a native drag event (dragover or drop) for a
  // single drop target, returning false if another target already claimed it. Drag
  // events bubble, so the innermost accepting target claims first and nested
  // targets do not multi-fire.
  claimDragEvent: (event: Event) => boolean;
  /** Shows `preview` at the pointer for the drag that `event` starts. */
  setPreview: (event: DragEvent, preview: ReactNode, anchor?: PreviewAnchor) => void;
  /**
   * A live ref to the current dragging state, readable at event time. It also holds a
   * drag that another window started.
   */
  stateRef: RefObject<DraggingState>;
  /** A live ref to the drag this window started. It is empty in every other window. */
  startedRef: RefObject<DraggingState>;
}

/**
 * Where a preview sits on the pointer. A `pointer` preview sits below and to the right
 * of the pointer. A `grab` preview keeps the point that the drag pressed on the source.
 * A `corner` preview puts its top-left corner on the pointer, where a canvas places a
 * dropped object.
 */
export type PreviewAnchor = "pointer" | "grab" | "corner";

// The preview layer's padding in px, which holds the preview's shadow. Keep it equal to
// the padding in Haul.css.
const PREVIEW_PAD = 18;
// Where a pointer-anchored preview starts, from the pointer.
const POINTER_OFFSET = { x: 14, y: 10 };
// Marks a source while it drags. Haul.css dims it.
const SOURCE_ATTRIBUTE = "data-haul-source";

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
  /**
   * The drag state. Set it, with `onChange`, to let the caller own it. Share one value
   * between the providers of several windows to let a drag cross between them.
   */
  value?: State;
  onChange?: (state: State) => void;
}

interface Started extends State {
  onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void;
}

/** Provides haul drag-and-drop state to descendants. */
export const Provider = memo(
  ({ children, value, onChange }: ProviderProps): ReactNode => {
    const ctx = useContext();

    const [dragging, setDragging] = state.usePurePassthrough({
      initialValue: ZERO_STATE,
      value,
      onChange,
    });
    // The drag this window started, until it ends.
    const ref = useRef<Started>(ZERO_STATE);
    const resolvedKey = useRef(dragging.key);
    const stateRef = useSyncedRef(dragging);
    const interceptors = useRef<Set<DragEndInterceptor>>(new Set());
    const watchers = useRef<Set<OnResolve>>(new Set());
    const handledEvents = useRef<WeakSet<Event>>(new WeakSet());
    const [preview, setPreviewContent] = useState<ReactNode>(null);
    const layerRef = useRef<HTMLDivElement>(null);

    // The browser takes one picture of the layer when dragstart returns, so the
    // preview renders at once and leaves before the next frame paints.
    const setPreview: ContextValue["setPreview"] = (event, content, anchor) => {
      flushSync(() => setPreviewContent(content));
      const layer = layerRef.current;
      if (layer == null) return;
      let x = PREVIEW_PAD - POINTER_OFFSET.x;
      let y = PREVIEW_PAD - POINTER_OFFSET.y;
      if (anchor === "corner") {
        x = PREVIEW_PAD;
        y = PREVIEW_PAD;
      } else if (anchor === "grab") {
        const rect = event.currentTarget.getBoundingClientRect();
        x = PREVIEW_PAD + event.clientX - rect.left;
        y = PREVIEW_PAD + event.clientY - rect.top;
      }
      event.dataTransfer.setDragImage(layer, x, y);
      requestAnimationFrame(() => flushSync(() => setPreviewContent(null)));
    };

    const claimDragEvent: ContextValue["claimDragEvent"] = (event) => {
      if (handledEvents.current.has(event)) return false;
      handledEvents.current.add(event);
      return true;
    };

    const resolve = (key: string, landed: boolean) => {
      if (resolvedKey.current === key) return;
      resolvedKey.current = key;
      watchers.current.forEach((watcher) => watcher(landed));
    };

    const start: ContextValue["start"] = (source, items, onSuccessfulDrop) => {
      const key = id.create();
      ref.current = { key, source, items, onSuccessfulDrop };
      setDragging({ key, source, items });
    };

    const finish = (key: string, result: OnSuccessfulDropProps | null) => {
      const started = ref.current;
      if (started.key === key) {
        ref.current = ZERO_STATE;
        if (result != null) started.onSuccessfulDrop?.(result);
      }
      resolve(key, result != null);
    };

    const cancel = (key: string) => {
      setDragging({ ...ZERO_STATE, key, drop: null });
      finish(key, null);
    };

    const drop: ContextValue["drop"] = ({ target, dropped }) => {
      // A drag that another window started reaches this one only in the shared state.
      const current =
        stateRef.current.items.length > 0 ? stateRef.current : ref.current;
      const props = { target, dropped, hauled: current.items };
      setDragging({ ...ZERO_STATE, key: current.key, drop: props });
      finish(current.key, props);
    };

    const end: ContextValue["end"] = (cursor, dropped = false) => {
      // A drag already finalized by a drop leaves nothing to end: dragend still fires
      // after it.
      if (ref.current.items.length === 0) return;
      let claimed: DropProps | null = null;
      interceptors.current.forEach((interceptor) => {
        if (claimed != null) return;
        claimed = interceptor(ref.current, cursor);
      });
      if (claimed != null) return drop(claimed);
      if (!dropped) cancel(ref.current.key);
    };

    // Only the shared state tells this window that another window ended a drag.
    useEffect(() => {
      if (dragging.drop !== undefined) finish(dragging.key, dragging.drop);
    }, [dragging]);

    // Only the window under the cursor sees a drop. A drop that no target took there
    // still ends the drag.
    useEffect(() => {
      const { key, items, drop } = dragging;
      if (items.length === 0 || drop !== undefined) return;
      const controller = new AbortController();
      window.addEventListener(
        "drop",
        (event) => {
          if (!event.defaultPrevented) cancel(key);
        },
        { signal: controller.signal },
      );
      return () => controller.abort();
    }, [dragging]);

    const bind: ContextValue["bind"] = (interceptor) => {
      interceptors.current.add(interceptor);
      return () => interceptors.current.delete(interceptor);
    };

    const watch: ContextValue["watch"] = (onResolve) => {
      watchers.current.add(onResolve);
      return () => watchers.current.delete(onResolve);
    };

    const controls: ContextValue = {
      start,
      end,
      drop,
      bind,
      watch,
      claimDragEvent,
      setPreview,
      stateRef,
      startedRef: ref,
    };
    // A nested Provider reuses the outer one's contexts, which are already in scope,
    // so it renders children without re-providing.
    if (ctx != null) return children;
    return (
      <Context value={controls}>
        <StateContext value={dragging}>{children}</StateContext>
        {createPortal(
          <div ref={layerRef} className={CSS.BE("haul", "layer")} aria-hidden>
            {preview}
          </div>,
          document.body,
        )}
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

/**
 * useOnResolve runs `onResolve` once per drag when it resolves, told whether the drag
 * landed on a drop target. Prefer it to watching {@link useDraggingState} clear: that
 * looks identical for a drop and for a cancelled drag, and it re-renders on every drag
 * in the window. `onResolve` need not be stable; the latest one always runs.
 */
export const useOnResolve = (onResolve: OnResolve): void => {
  const ctx = useContext();
  const ref = useSyncedRef(onResolve);
  useEffect(() => ctx?.watch((landed) => ref.current(landed)), [ctx, ref]);
};

/** Passed to a drag's `onSuccessfulDrop` callback once a drop completes. */
export interface OnSuccessfulDropProps {
  target: Item;
  hauled: Item[];
  dropped: Item[];
}

/** Props for {@link useDrag}; `key` defaults to a generated ID. */
export interface UseDragProps extends optional.Optional<Item, "key"> {}

/** Arguments to {@link UseDragReturn.startDrag}. */
export interface StartDragProps {
  /** The source's `dragstart` event. */
  event: DragEvent;
  items: Item[];
  /**
   * The picture that follows the pointer. It renders outside the source's contexts, so
   * it takes plain values.
   */
  preview: ReactNode;
  anchor?: PreviewAnchor;
  onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void;
}

// The source dims until the drag ends. A selected tree item drags its whole selection,
// so every selected item dims.
const dimSources = (pressed: Element): void => {
  const tree = pressed.closest('[role="tree"]');
  const sources =
    tree != null && pressed.getAttribute("aria-selected") === "true"
      ? [...tree.querySelectorAll('[role="treeitem"][aria-selected="true"]')]
      : [pressed];
  sources.forEach((el) => el.setAttribute(SOURCE_ATTRIBUTE, ""));
  pressed.addEventListener(
    "dragend",
    () => sources.forEach((el) => el.removeAttribute(SOURCE_ATTRIBUTE)),
    { once: true },
  );
};

/** Return value of {@link useDrag}. */
export interface UseDragReturn {
  /** Begins hauling items and shows their preview at the pointer. */
  startDrag: (props: StartDragProps) => void;
  /** Native `dragend`/`mouseup` handler that resolves the drag. */
  onDragEnd: (e: DragEvent | MouseEvent) => void;
}

/** Wires an element up as a drag source of `type`. */
export const useDrag = ({ type, key }: UseDragProps): UseDragReturn => {
  const generatedKey = useId();
  const source: Item = { key: key ?? generatedKey, type };
  const ctx = useContext();
  const startDrag: UseDragReturn["startDrag"] = ({
    event,
    items,
    preview,
    anchor,
    onSuccessfulDrop,
  }) => {
    if (ctx == null) return;
    const { start, end, startedRef, setPreview } = ctx;
    start(source, items, onSuccessfulDrop);
    setPreview(event, preview, anchor);
    dimSources(event.currentTarget);
    // A source that unmounts mid-drag fires `dragend` on a detached node, which never
    // reaches here. Mouse events resume only once a drag is over, so the first one ends
    // a drag its own source cannot.
    document.addEventListener(
      "mousemove",
      (e) => {
        if (startedRef.current.items.length === 0) return;
        end(xy.construct({ x: e.screenX, y: e.screenY }));
      },
      { once: true },
    );
  };
  // Dragend reports a drop effect when a drop happened, maybe in another window.
  const onDragEnd = (e: DragEvent | MouseEvent) =>
    ctx?.end(
      xy.construct({ x: e.screenX, y: e.screenY }),
      "dataTransfer" in e && e.dataTransfer.dropEffect !== "none",
    );
  return { startDrag, onDragEnd };
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
  const generatedKey = useId();
  const target: Item = { key: key ?? generatedKey, type };
  const prevDragOver = useRef<xy.XY>({ x: -1, y: -1 });

  const handleDragOver = (event: DragEvent) => {
    if (ctx == null) return;
    event.preventDefault();
    const cursor = xy.construct({ x: event.screenX, y: event.screenY });
    if (xy.equals(cursor, prevDragOver.current)) return;
    prevDragOver.current = cursor;
    if (!canDrop(ref.current)) return;
    if (!ctx.claimDragEvent(event.nativeEvent)) return;
    onDragOver?.({ event, ...ref.current });
  };

  const handleDrop = (event: DragEvent) => {
    if (ctx == null) return;
    if (!canDrop(ref.current)) return;
    if (!ctx.claimDragEvent(event.nativeEvent)) return;
    event.preventDefault();
    ctx.drop({ target, dropped: onDrop({ ...ref.current, event }) });
  };

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
  const sourceAndTarget: Item = { key: key ?? generatedKey, type };
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

/** Props for a drop handled outside the normal tree; `onDrop` also gets the cursor. */
export interface UseDropOutsideProps extends Omit<UseDropProps, "onDrop"> {
  onDrop: (props: OnDropProps, cursor: xy.XY) => Item[];
}
