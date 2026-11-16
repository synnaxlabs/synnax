// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { type Component } from "@/component";
import { context } from "@/context";
import { SubmenusContext, useSubmenus } from "@/menu/submenus";
import { type Theming } from "@/theming";

export interface ContextValue {
  onClick: (key: string) => void;
  selected: string;
  level?: text.Level;
  gap?: Component.Size;
  background?: Theming.Shade;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { onClick: () => {}, selected: "" },
  displayName: "Menu.Context",
});
export { useContext };

/** Props for {@link Menu}. */
export interface MenuProps
  extends PropsWithChildren, Pick<ContextValue, "level" | "gap" | "background"> {
  /** The key of the selected item. */
  value?: string;
  /**
   * Called with the key of the clicked item. Pass a record instead to route each key to
   * its own handler.
   */
  onChange?: ((key: string) => void) | Record<string, (key: string) => void>;
}

/**
 * Holds the shared state for a list of {@link Item}s: which one is selected, what
 * happens on click, the text level, gap, and background they inherit, and which of its
 * {@link Sub}s is open. It renders no element of its own.
 *
 * @example
 * <Menu.Menu onChange={{ rename, delete: del }}>
 *   <Menu.Item itemKey="rename">Rename</Menu.Item>
 * </Menu.Menu>
 */
export const Menu = ({
  children,
  onChange,
  level,
  gap,
  background,
  value: selected = "",
}: MenuProps): ReactElement => {
  const submenus = useSubmenus();
  const onClick = (key: string) => {
    if (typeof onChange === "function") onChange(key);
    else if (onChange && key in onChange) onChange[key](key);
  };
  const ctxValue = {
    onClick,
    selected,
    level,
    gap,
    background,
  };
  return (
    <Context value={ctxValue}>
      <SubmenusContext value={submenus}>{children}</SubmenusContext>
    </Context>
  );
};
