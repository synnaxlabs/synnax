// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/haul/Haul.css";

import {
  box,
  Destructor,
  type Key,
  type Optional,
  UnknownRecord,
  xy,
} from "@synnaxlabs/x";
import React, {
  createContext,
  type DragEvent,
  type DragEventHandler,
  type MutableRefObject,
  type PropsWithChildren,
  useCallback,
  useContext as reactUseContext,
  useEffect,
  useId,
  useMemo,
  useRef,
} from "react";
import { z } from "zod";

import { type state } from "@/state";

export const itemZ = z.object({
  key: z.string().or(z.number()).or(z.symbol()),
  type: z.string(),
  elementID: z.string().optional(),
  data: z.record(z.unknown()).optional(),
});

// Item represents a draggable item.
export interface Item {
  key: Key;
  type: string;
  elementID?: string;
  data?: UnknownRecord;
}

export const draggingStateZ = z.object({
  source: itemZ,
  items: z.array(itemZ),
});

export interface DraggingState {
  source: Item;
  items: Item[];
}

const ZERO_ITEM: Item = { key: "", type: "" };

export const ZERO_DRAGGING_STATE: DraggingState = {
  source: ZERO_ITEM,
  items: [],
};

interface DropProps {
  target: Item;
  dropped: Item[];
}

export const FILE_TYPE = "file";

type DragEndInterceptor = (state: DraggingState, cursor: xy.XY) => DropProps | null;
export interface ContextValue {
  state: DraggingState;
  start: (
    source: Item,
    items: Item[],
    onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void,
  ) => void;
  end: (cursor: xy.XY) => void;
  drop: (props: DropProps) => void;
  bind: (interceptor: DragEndInterceptor) => Destructor;
}

const Context = createContext<ContextValue | null>(null);

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

export const useContext = (): ContextValue | null => reactUseContext(Context);

export const Provider = ({
  children,
  useState = React.useState,
  onDropOutside,
}: ProviderProps): JSX.Element => {
  const ctx = reactUseContext(Context);

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
  return <Context.Provider value={oCtx}>{children}</Context.Provider>;
};

// |||||| DRAGGING ||||||

export const useDraggingRef = (): MutableRefObject<DraggingState> => {
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

export interface UseDragProps extends Optional<Item, "key"> {}

export interface UseDragReturn {
  startDrag: (
    items: Item[],
    onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void,
  ) => void;
  onDragEnd: (e: DragEvent) => void;
}

export const useDrag = ({ type, key }: UseDragProps): UseDragReturn => {
  const key_ = key ?? useId();
  const source: Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const ctx = useContext();
  if (ctx == null) return { startDrag: () => {}, onDragEnd: () => {} };
  const { start, end } = ctx;
  return {
    startDrag: useCallback((items, f) => start(source, items, f), [start, source]),
    onDragEnd: (e: DragEvent) => end(xy.construct({ x: e.screenX, y: e.screenY })),
  };
};

// |||||| DROP ||||||

export type CanDrop = (state: DraggingState) => boolean;

export interface OnDropProps extends DraggingState {
  event?: DragEvent;
}

export type OnDragOverProps = OnDropProps;

export interface UseDropProps extends Optional<Item, "key"> {
  canDrop: CanDrop;
  onDrop: (props: OnDropProps) => Item[];
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

  const handleDragOver = useCallback(
    (event: DragEvent) => {
      if (!canDrop(ref.current)) return;
      event.preventDefault();
      onDragOver?.({
        event,
        ...ref.current,
      });
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
  extends Omit<UseDragProps, "source">,
    Omit<UseDropProps, "target">,
    Optional<Item, "key"> {}

export interface UseDragAndDropReturn extends UseDragReturn, UseDropReturn {}

export const useDragAndDrop = ({
  type,
  key,
  ...props
}: UseDragAndDropProps): UseDragAndDropReturn => {
  const key_ = key ?? useId();
  const sourceAndTarget: Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const dragProps = useDrag(sourceAndTarget);
  const dropProps = useDrop({ ...props, ...sourceAndTarget });
  return { ...dragProps, ...dropProps };
};

export const canDropOfType =
  (type: string): CanDrop =>
  ({ items }) =>
    items.some((entity) => entity.type === type);

export const filterByType = (type: string, entities: Item[]): Item[] =>
  entities.filter((entity) => entity.type === type);

export interface UseDropOutsideProps extends Omit<UseDropProps, "onDrop"> {
  onDrop: (props: OnDropProps, cursor: xy.XY) => Item[];
}

export const useDropOutside = ({ type, key, ...rest }: UseDropOutsideProps): void => {
  const ctx = useContext();
  if (ctx == null) return;
  const dragging = useDraggingRef();
  const { bind } = ctx;
  const isOutside = useRef(false);
  const key_ = key ?? useId();
  const target: Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const propsRef = useRef<UseDropOutsideProps>({ ...rest, type, key: key_ });
  useEffect(() => {
    const release = bind((state, cursor) => {
      const { canDrop, onDrop } = propsRef.current;
      if (!canDrop(state) || !isOutside.current) return null;
      const dropped = onDrop({ ...state }, cursor);
      return { target, dropped };
    });
    const handleMouseEnter = () => {
      const { canDrop } = propsRef.current;
      isOutside.current = false;
      if (!canDrop(dragging.current)) return;
    };
    const handleMouseLeave = (e: globalThis.DragEvent) => {
      const { onDragOver, canDrop } = propsRef.current;
      const windowBox = box.construct(window.document.documentElement);
      if ((box.contains(windowBox, xy.construct(e.clientX, e.clientY)), false)) return;
      isOutside.current = true;
      if (!canDrop(dragging.current)) return;
      onDragOver?.(dragging.current);
    };
    document.body.addEventListener("dragleave", handleMouseLeave);
    document.body.addEventListener("mouseenter", handleMouseEnter);
    return () => {
      release();
      document.body.removeEventListener("dragleave", handleMouseLeave);
      document.body.removeEventListener("mouseenter", handleMouseEnter);
    };
  }, []);
};
