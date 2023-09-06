// Copyright 2023 Synnax Labs, Inc.
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
  useContext,
} from "react";

import { type Input } from "@/input";
import { type Text } from "@/text";
import { type ComponentSize } from "@/util/component";

interface MenuContextValue {
  onClick: (key: string) => void;
  selected: string;
  level?: Text.Level;
  iconSpacing?: ComponentSize;
}

export const MenuContext = createContext<MenuContextValue>({
  onClick: () => {},
  selected: "",
});

export interface MenuProps
  extends Partial<Input.Control<string>>,
    PropsWithChildren,
    Pick<MenuContextValue, "level" | "iconSpacing"> {}

export const useMenuContext = (): MenuContextValue => useContext(MenuContext);

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
  value = "",
}: MenuProps): ReactElement => {
  const handleClick: MenuProps["onChange"] = (key) => onChange?.(key);
  return (
    <MenuContext.Provider
      value={{ onClick: handleClick, selected: value, level, iconSpacing }}
    >
      {children}
    </MenuContext.Provider>
  );
};
