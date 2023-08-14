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
  useContext,
  useMemo,
  useRef,
} from "react";

import { Key } from "@synnaxlabs/x";

import { PureUseState } from "@/util/state";

import "@/core/haul/Haul.css";

export interface Hauled {
  key: Key;
  type: string;
}

export interface HaulContextValue {
  value: Hauled[];
  start: (hauled: Hauled[], onSuccessfulDrop?: (hauled: Hauled[]) => void) => void;
  end: () => void;
  drop: () => void;
}

const HaulContext = createContext<HaulContextValue | null>(null);

export interface HaulProviderProps extends PropsWithChildren {
  useState?: PureUseState<Hauled[]>;
}

interface HaulProviderRef {
  dragging: Hauled[];
  onSuccessfulDrop: (hauled: Hauled[]) => void;
}

const ZERO_HAUL_REF: HaulProviderRef = { dragging: [], onSuccessfulDrop: () => {} };

export const HaulProvider = ({
  children,
  useState = React.useState,
}: HaulProviderProps): JSX.Element => {
  const ctx = useContext(HaulContext);

  const [value, onChange] = useState();

  const ref = useRef<HaulProviderRef>(ZERO_HAUL_REF);
  const start = useCallback(
    (hauled: Hauled[], onSuccessfulDrop?: (hauled: Hauled[]) => void) => {
      ref.current.dragging = hauled;
      if (onSuccessfulDrop != null) ref.current.onSuccessfulDrop = onSuccessfulDrop;
      onChange(hauled);
    },
    [onChange]
  );
  const end = useCallback(() => {
    ref.current = ZERO_HAUL_REF;
    onChange([]);
  }, [onChange]);

  const drop = useCallback(() => {
    ref.current.onSuccessfulDrop(ref.current.dragging);
    ref.current = ZERO_HAUL_REF;
    onChange([]);
  }, [onChange]);

  const oCtx = useMemo<HaulContextValue>(
    () => ctx ?? { value, start, end, drop },
    [value, start, end, drop, ctx]
  );
  return <HaulContext.Provider value={oCtx}>{children}</HaulContext.Provider>;
};

export const useHaulContext = (): HaulContextValue => {
  const ctx = useContext(HaulContext);
  if (ctx == null) throw new Error("HaulContext not available");
  return ctx;
};

export const useHaulDraggingRef = (): MutableRefObject<Hauled[]> => {
  const ref = useRef<Hauled[]>([]);
  const { value } = useHaulContext();
  ref.current = value;
  return ref;
};

export const useHaulDraggingState = (): Hauled[] => {
  const { value } = useHaulContext();
  return value;
};

export interface UseHaulDragProps {}

export interface UseHaulDragReturn {
  startDrag: (
    entities: Hauled[],
    onSuccessfulDrop?: (hauled: Hauled[]) => void
  ) => void;
  endDrag: () => void;
}

export const useHaulDrag = (): UseHaulDragReturn => {
  const { start, end } = useHaulContext();
  return { startDrag: start, endDrag: end };
};

export interface UseHaulDropProps {
  canDrop: (entities: Hauled[]) => boolean;
  onDrop: (entities: Hauled[], e: DragEvent) => void;
  onDragOver?: (entities: Hauled[], e: DragEvent) => void;
}

export interface UseHaulDropReturn {
  onDragOver: DragEventHandler;
  onDrop: DragEventHandler;
}

export const useHaulDrop = ({
  canDrop,
  onDrop,
  onDragOver,
}: UseHaulDropProps): UseHaulDropReturn => {
  const ref = useHaulDraggingRef();
  const { drop } = useHaulContext();

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
