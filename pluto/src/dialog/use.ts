// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  use as reactUse,
  useCallback,
  useEffect,
  useState,
} from "react";

/** Props for the {@link use} hook. */
export interface UseProps {
  initialVisible?: boolean;
  onVisibleChange?: (vis: boolean) => void;
}

/** Return type for the {@link use} hook. */
export interface UseReturn {
  visible: boolean;
  close: () => void;
  open: () => void;
  toggle: (vis?: boolean | unknown) => void;
}

export interface ContextValue extends Pick<UseReturn, "close"> {}

const Context = createContext<ContextValue>({
  close: () => {
    console.error("Dialog context not provided.");
  },
});

export const Provider = Context;

export const useContext = () => reactUse(Context);

/**
 * Implements basic dropdown behavior, and should be preferred when using
 * the {@link Dialog} component. Opens the dropdown whenever the 'open' function is
 * called, and closes it whenever the 'close' function is called OR the user clicks
 * outside of the dropdown parent wrapped,which includes the dropdown trigger (often
 * a button or input).
 *
 * @param initialVisible - Whether the dropdown should be visible on mount.
 * @returns visible - Whether the dropdown is visible.
 * @returns close - A function to close the dropdown.
 * @returns open - A function to open the dropdown.
 * @returns toggle - A function to toggle the dropdown.
 */
export const use = (props?: UseProps): UseReturn => {
  const { initialVisible = false, onVisibleChange } = props ?? {};
  const [visible, setVisible] = useState(initialVisible);
  useEffect(() => onVisibleChange?.(visible), [visible, onVisibleChange]);
  const toggle = useCallback(
    (vis?: boolean | unknown) =>
      setVisible((v) => {
        if (typeof vis === "boolean") return vis;
        return !v;
      }),
    [setVisible, onVisibleChange],
  );
  const open = useCallback(() => toggle(true), [toggle]);
  const close = useCallback(() => toggle(false), [toggle]);
  return { visible, open, close, toggle };
};
