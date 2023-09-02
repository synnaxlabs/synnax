// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, {
  type DragEvent,
  type DragEventHandler,
  type MutableRefObject,
  type PropsWithChildren,
  createContext,
  useCallback,
  useContext as reactUseContext,
  useMemo,
  useRef,
  useId,
} from "react";

import { type Key, type Optional } from "@synnaxlabs/x";

import { type state } from "@/state";

import "@/haul/Haul.css";

// Item represents a draggable item.
export interface Item {
  key: Key;
  type: string;
}

export interface DraggingState {
  source: Item;
  items: Item[];
}

const ZERO_ITEM: Item = { key: "", type: "" };

export const ZERO_DRAGGING_STATE: DraggingState = {
  source: ZERO_ITEM,
  items: [],
};

export interface ContextValue {
  state: DraggingState;
  start: (
    source: Item,
    items: Item[],
    onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void,
  ) => void;
  end: () => void;
  drop: (target: Item, dropped: Item[]) => void;
}

const Context = createContext<ContextValue | null>(null);

export interface ProviderProps extends PropsWithChildren {
  useState?: state.PureUse<DraggingState>;
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
}: ProviderProps): JSX.Element => {
  const ctx = reactUseContext(Context);

  const [state, setState] = useState(ZERO_DRAGGING_STATE);
  const ref = useRef<ProviderRef>(HAUL_REF);

  const start: ContextValue["start"] = useCallback(
    (source, items, onSuccessfulDrop) => {
      ref.current = { source, items, onSuccessfulDrop };
      setState({ source, items });
    },
    [setState],
  );

  const end: ContextValue["end"] = useCallback(() => {
    ref.current = HAUL_REF;
    setState(ZERO_DRAGGING_STATE);
  }, [setState]);

  const drop: ContextValue["drop"] = useCallback(
    (target, dropped) => {
      ref.current.onSuccessfulDrop?.({
        target,
        dropped,
        hauled: ref.current.items,
      });
      ref.current = HAUL_REF;
      setState(ZERO_DRAGGING_STATE);
    },
    [setState],
  );

  const oCtx = useMemo<ContextValue>(
    () => ctx ?? { state, start, end, drop },
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
  onDragEnd: () => void;
}

export const useDrag = ({ type, key }: UseDragProps): UseDragReturn => {
  const key_ = key ?? useId();
  const source: Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const ctx = useContext();
  if (ctx == null) return { startDrag: () => {}, onDragEnd: () => {} };
  const { start, end } = ctx;
  return {
    startDrag: useCallback((items, f) => start(source, items, f), [start, source]),
    onDragEnd: end,
  };
};

// |||||| DROP ||||||

export type CanDrop = (state: DraggingState) => boolean;

export interface OnDropProps extends DraggingState {
  event: DragEvent;
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
      drop(target, onDrop({ ...ref.current, event }));
    },
    [ref, onDrop, canDrop, drop, target],
  );

  return {
    onDragOver: handleDragOver,
    onDrop: handleDrop,
  };
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
