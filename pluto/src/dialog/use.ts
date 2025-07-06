// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, useCallback, useEffect, useMemo } from "react";

import { useCombinedStateAndRef, useRequiredContext, useSyncedRef } from "@/hooks";
import { state } from "@/state";

/** Props for the {@link use} hook. */
export interface UseProps {
  /**
   * Whether the dialog should be visible on mount.
   */
  initialVisible?: boolean;
  /**
   * A callback invoked whenever the dialog's visibility changes.
   *
   * @param visible - The new visibility state.
   */
  onVisibleChange?: (visible: boolean) => void;
}

/** Return type for the {@link use} hook. */
export interface UseReturn {
  /**
   * Function to close the dialog.
   */
  close: () => void;
  /**
   * Function to open the dialog.
   */
  open: () => void;
  /**
   * Function to toggle the dialog.
   */
  toggle: () => void;
  /**
   * Whether the dialog is currently visible.
   */
  visible: boolean;
}

export interface ContextValue extends Pick<UseReturn, "close" | "open"> {}

const Context = createContext<ContextValue | null>(null);

export const Provider = Context;

export const useContext = (): ContextValue => useRequiredContext(Context);

/**
 * Implements basic dialog behavior. Opens the dialog whenever the 'open' function is
 * called, closes it whenever the 'close' function is called, and toggles it whenever
 * the 'toggle' function is called.
 *
 * @param initialVisible - Whether the dialog should be visible on mount.
 * @param onVisibleChange - A function to call whenever the visibility of the dialog
 * changes.
 * @returns close - A function to close the dialog.
 * @returns open - A function to open the dialog.
 * @returns toggle - A function to toggle the dialog.
 * @returns visible - Whether the dialog is visible.
 */
export const use = ({
  initialVisible = false,
  onVisibleChange,
}: UseProps = {}): UseReturn => {
  const [visible, setVisible, visibleRef] = useCombinedStateAndRef(initialVisible);
  const onVisibleChangeRef = useSyncedRef(onVisibleChange);
  useEffect(() => onVisibleChangeRef.current?.(visible), [visible]);
  const set = useCallback((setter: state.SetArg<boolean>) => {
    const nextVisible = state.executeSetter(setter, visibleRef.current);
    onVisibleChangeRef.current?.(nextVisible);
    setVisible(nextVisible);
  }, []);
  return useMemo(
    () => ({
      close: () => set(false),
      open: () => set(true),
      toggle: () => set((prev) => !prev),
      visible,
    }),
    [set, visible],
  );
};
