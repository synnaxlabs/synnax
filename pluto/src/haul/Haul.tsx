// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/haul/Haul.css";

import { type destructor, type optional, record, xy } from "@synnaxlabs/x";
import React, {
  type DragEvent,
  type DragEventHandler,
  memo,
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  useCallback,
  useId,
  useMemo,
  useRef,
} from "react";
import { z } from "zod";

import { context } from "@/context";
import { type state } from "@/state";

export const itemZ = z.object({
  key: z.string().or(z.number()),
  type: z.string(),
  elementID: z.string().optional(),
  data: record.unknownZ.optional(),
});

// Item represents a draggable item.
export interface Item {
  key: record.Key;
  type: string;
  elementID?: string;
  data?: record.Unknown;
}

export const draggingStateZ = z.object({ source: itemZ, items: z.array(itemZ) });

export interface DraggingState {
  source: Item;
  items: Item[];
}

export const ZERO_ITEM: Item = { key: "", type: "" };

export const ZERO_DRAGGING_STATE: DraggingState = { source: ZERO_ITEM, items: [] };

interface DropProps {
  target: Item;
  dropped: Item[];
}

export const FILE_TYPE = "file";

export const FILE: Item = { key: "file", type: FILE_TYPE };

// Effects that indicate a file is being dragged. Downside is that this also
// allows dragging links, but that's not a huge deal.
const ALLOWED_FILE_DRAG_EFFECTS = new Set(["all", "copyLink"]);

export const isFileDrag = (event: DragEvent, dragging: DraggingState): boolean =>
  ALLOWED_FILE_DRAG_EFFECTS.has(event.dataTransfer.effectAllowed) &&
  dragging.items.length === 0;

interface DragEndInterceptor {
  (state: DraggingState, cursor: xy.XY): DropProps | null;
}

export interface ContextValue {
  state: DraggingState;
  start: (
    source: Item,
    items: Item[],
    onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void,
  ) => void;
  end: (cursor: xy.XY) => void;
  drop: (props: DropProps) => void;
  bind: (interceptor: DragEndInterceptor) => destructor.Destructor;
}

const [Context, useContext] = context.create<ContextValue | null>({
  defaultValue: null,
  displayName: "Haul.Context",
});
export { useContext };

export interface ProviderProps extends PropsWithChildren {
  useState?: state.PureUse<DraggingState>;
  onDropOutside?: (props: OnDropProps) => Item[];
}

interface ProviderRef extends DraggingState {
  onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void;
}

const HAUL_REF: ProviderRef = {
  ...ZERO_DRAGGING_STATE,
  onSuccessfulDrop: () => {},
};

export const Provider = memo(
  ({
    children,
    useState = React.useState,
    onDropOutside,
  }: ProviderProps): ReactElement => {
    const ctx = useContext();

    const [state, setState] = useState(ZERO_DRAGGING_STATE);
    const ref = useRef<ProviderRef>(HAUL_REF);
    const interceptors = useRef<Set<DragEndInterceptor>>(new Set());

    const start: ContextValue["start"] = useCallback(
      (source, items, onSuccessfulDrop) => {
        ref.current = { source, items, onSuccessfulDrop };
        setState({ source, items });
      },
      [setState, onDropOutside],
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

    const oCtx = useMemo<ContextValue>(
      () => ctx ?? { state, start, end, drop, bind },
      [state, start, end, drop, ctx],
    );
    return <Context value={oCtx}>{children}</Context>;
  },
);
Provider.displayName = "HaulProvider";

// |||||| DRAGGING ||||||

export const useDraggingRef = (): RefObject<DraggingState> => {
  const ref = useRef<DraggingState>(ZERO_DRAGGING_STATE);
  const ctx = useContext();
  if (ctx == null) return ref;
  ref.current = ctx.state;
  return ref;
};

export const useDraggingState = (): DraggingState => {
  const ctx = useContext();
  if (ctx == null) return ZERO_DRAGGING_STATE;
  return ctx.state;
};

// |||||| DRAG ||||||

export interface OnSuccessfulDropProps {
  target: Item;
  hauled: Item[];
  dropped: Item[];
}

export interface UseDragProps extends optional.Optional<Item, "key"> {}

export interface UseDragReturn {
  startDrag: (
    items: Item[],
    onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void,
  ) => void;
  onDragEnd: (e: DragEvent | MouseEvent) => void;
}

export const useDrag = ({ type, key }: UseDragProps): UseDragReturn => {
  const key_ = key ?? useId();
  const source: Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const ctx = useContext();
  if (ctx == null) return { startDrag: () => {}, onDragEnd: () => {} };
  const { start, end } = ctx;
  return {
    startDrag: useCallback((items, f) => start(source, items, f), [start, source]),
    onDragEnd: (e: DragEvent | MouseEvent) =>
      end(xy.construct({ x: e.screenX, y: e.screenY })),
  };
};

// |||||| DROP ||||||

export interface CanDrop {
  (state: DraggingState): boolean;
}

export interface OnDropProps extends DraggingState {
  event?: DragEvent;
}

export interface OnDrop {
  (props: OnDropProps): Item[];
}

export interface OnDragOverProps extends OnDropProps {}

export interface UseDropProps extends optional.Optional<Item, "key"> {
  canDrop: CanDrop;
  onDrop: OnDrop;
  onDragOver?: (props: OnDragOverProps) => void;
}

export interface UseDropReturn {
  onDragOver: DragEventHandler;
  onDrop: DragEventHandler;
}

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
  const { drop } = ctx;

  const key_ = key ?? useId();
  const target: Item = useMemo(() => ({ key: key_, type }), [key_, type]);

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
      event.preventDefault();
      drop({ target, dropped: onDrop({ ...ref.current, event }) });
    },
    [ref, onDrop, canDrop, drop, target],
  );

  return { onDragOver: handleDragOver, onDrop: handleDrop };
};

// |||||| DRAG AND DROP ||||||

export interface UseDragAndDropProps
  extends
    Omit<UseDragProps, "source">,
    Omit<UseDropProps, "target">,
    optional.Optional<Item, "key"> {}

export interface UseDragAndDropReturn extends UseDragReturn, UseDropReturn {}

export const useDragAndDrop = ({
  type,
  key,
  ...rest
}: UseDragAndDropProps): UseDragAndDropReturn => {
  const key_ = key ?? useId();
  const sourceAndTarget: Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const dragProps = useDrag(sourceAndTarget);
  const dropProps = useDrop({ ...rest, ...sourceAndTarget });
  return { ...dragProps, ...dropProps };
};

export const canDropOfType =
  (type: string): CanDrop =>
  ({ items }) =>
    items.some((entity) => entity.type === type);

export const filterByType = (type: string, entities: Item[]): Item[] =>
  entities.filter((entity) => entity.type === type);

export const useFilterByTypeCallback = (
  type: string,
  fn: OnDrop,
  deps: unknown[],
): OnDrop =>
  useCallback(
    (props) => fn({ ...props, items: filterByType(type, props.items) }),
    deps,
  );

export interface UseDropOutsideProps extends Omit<UseDropProps, "onDrop"> {
  onDrop: (props: OnDropProps, cursor: xy.XY) => Item[];
}
