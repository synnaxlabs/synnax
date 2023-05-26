// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, PropsWithChildren, useContext } from "react";

import { InputControl } from "@/core/Input";

interface MenuContextValue {
  onClick: (key: string) => void;
  selected: string;
}

export const MenuContext = createContext<MenuContextValue>({
  onClick: () => {},
  selected: "",
});

export interface MenuProps extends Partial<InputControl<string>>, PropsWithChildren {}

export const useMenuContext = (): MenuContextValue => useContext(MenuContext);

export const Menu = ({ children, onChange, value = "" }: MenuProps): ReactElement => {
  const handleClick: MenuProps["onChange"] = (key) => onChange?.(key);
  return (
    <MenuContext.Provider value={{ onClick: handleClick, selected: value }}>
      {children}
    </MenuContext.Provider>
  );
};
