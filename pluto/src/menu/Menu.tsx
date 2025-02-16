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
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useMemo,
} from "react";

import { Divider as CoreDivider } from "@/divider";
import { type Text } from "@/text";
import { type ComponentSize } from "@/util/component";

export interface ContextValue {
  onClick: (key: string) => void;
  selected: string;
  level?: Text.Level;
  iconSpacing?: ComponentSize;
}

const Context = createContext<ContextValue>({ onClick: () => {}, selected: "" });

export interface MenuProps
  extends PropsWithChildren,
    Pick<ContextValue, "level" | "iconSpacing"> {
  value?: string;
  onChange?: ((key: string) => void) | Record<string, (key: string) => void>;
}

export const useContext = () => use(Context);

/**
 * Menu is a modular component that allows you to create a menu with a list of items.
 * It satisfies the InputControl string interface, so it's selected value can be
 * controlled.
 *
 * @param props - Props for the component. All unlisted props will be spread to the
 * underlying Space component acting as the root element.
 * @param props.onChange - Callback executed when the selected item changes.
 * @param props.value - The selected item.
 */
export const Menu = ({
  children,
  onChange,
  level,
  iconSpacing,
  value: selected = "",
}: MenuProps): ReactElement => {
  const onClick = useCallback(
    (key: string) => {
      if (typeof onChange === "function") onChange(key);
      else if (onChange && key in onChange) onChange[key](key);
    },
    [onChange],
  );
  const ctxValue = useMemo(
    () => ({
      onClick,
      selected,
      level,
      iconSpacing,
    }),
    [selected, onClick, level, iconSpacing],
  );
  return <Context value={ctxValue}>{children}</Context>;
};

export const Divider = (): ReactElement => <CoreDivider.Divider direction="x" padded />;
