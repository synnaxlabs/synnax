// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, {
  DragEvent,
  DragEventHandler,
  MutableRefObject,
  PropsWithChildren,
  createContext,
  useCallback,
  useContext as reactUseContext,
  useMemo,
  useRef,
} from "react";

import { Key } from "@synnaxlabs/x";

import { PureUseState } from "@/util/state";

import "@/haul/Haul.css";

export interface Item {
  key: Key;
  type: string;
}

export interface ContextValue {
  value: Item[];
  start: (hauled: Item[], onSuccessfulDrop?: (hauled: Item[]) => void) => void;
  end: () => void;
  drop: () => void;
}

const Context = createContext<ContextValue | null>(null);

export interface ProviderProps extends PropsWithChildren {
  useState?: PureUseState<Item[]>;
}

interface ProviderRef {
  dragging: Item[];
  onSuccessfulDrop: (hauled: Item[]) => void;
}

const HAUL_REF: ProviderRef = { dragging: [], onSuccessfulDrop: () => {} };

export const useContext = (): ContextValue => {
  const ctx = reactUseContext(Context);
  if (ctx == null) throw new Error("HaulContext not available");
  return ctx;
};

export const Provider = ({
  children,
  useState = React.useState,
}: ProviderProps): JSX.Element => {
  const ctx = useContext();

  const [value, onChange] = useState();

  const ref = useRef<ProviderRef>(HAUL_REF);
  const start = useCallback(
    (hauled: Item[], onSuccessfulDrop?: (hauled: Item[]) => void) => {
      ref.current.dragging = hauled;
      if (onSuccessfulDrop != null) ref.current.onSuccessfulDrop = onSuccessfulDrop;
      onChange(hauled);
    },
    [onChange]
  );
  const end = useCallback(() => {
    ref.current = HAUL_REF;
    onChange([]);
  }, [onChange]);

  const drop = useCallback(() => {
    ref.current.onSuccessfulDrop(ref.current.dragging);
    ref.current = HAUL_REF;
    onChange([]);
  }, [onChange]);

  const oCtx = useMemo<ContextValue>(
    () => ctx ?? { value, start, end, drop },
    [value, start, end, drop, ctx]
  );
  return <Context.Provider value={oCtx}>{children}</Context.Provider>;
};

export const useDraggingRef = (): MutableRefObject<Item[]> => {
  const ref = useRef<Item[]>([]);
  const { value } = useContext();
  ref.current = value;
  return ref;
};

export const useDraggingState = (): Item[] => {
  const { value } = useContext();
  return value;
};

export interface UseDragReturn {
  startDrag: (entities: Item[], onSuccessfulDrop?: (hauled: Item[]) => void) => void;
  endDrag: () => void;
}

export const useDrag = (): UseDragReturn => {
  const { start, end } = useContext();
  return { startDrag: start, endDrag: end };
};

type CanDrop = (entities: Item[]) => boolean;

export interface UseDropProps {
  canDrop: CanDrop;
  onDrop: (entities: Item[], e: DragEvent) => void;
  onDragOver?: (entities: Item[], e: DragEvent) => void;
}

export interface UseDropReturn {
  onDragOver: DragEventHandler;
  onDrop: DragEventHandler;
}

export const useDrop = ({
  canDrop,
  onDrop,
  onDragOver,
}: UseDropProps): UseDropReturn => {
  const ref = useDraggingRef();
  const { drop } = useContext();

  const handleDragOver = useCallback(
    (e: DragEvent) => {
      if (!canDrop(ref.current)) return;
      e.preventDefault();
      onDragOver?.(ref.current, e);
    },
    [ref, canDrop]
  );

  const handleDrop = useCallback(
    (e: DragEvent) => {
      if (!canDrop(ref.current)) return;
      e.preventDefault();
      drop();
      onDrop(ref.current, e);
    },
    [ref, onDrop, canDrop, drop]
  );

  return {
    onDragOver: handleDragOver,
    onDrop: handleDrop,
  };
};

export const canDropOfType =
  (type: string): CanDrop =>
  (entities) =>
    entities.some((entity) => entity.type === type);
