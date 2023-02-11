// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, PropsWithChildren, useContext } from "react";

import clsx from "clsx";

import { Button, ButtonProps } from "@/core/Button";
import { Space } from "@/core/Space";

interface CoreMenuContextProps {
  onClick: (key: string) => void;
}

export const CoreMenuContext = createContext<CoreMenuContextProps>({
  onClick: () => {},
});

export interface MenuProps extends PropsWithChildren {
  onClick?: (key: string) => void;
}

const useCoreMenuContext = (): CoreMenuContextProps => useContext(CoreMenuContext);

export const Menu = ({ children, onClick }: MenuProps): JSX.Element => {
  const handleClick = (key: string): void => {
    onClick?.(key);
  };
  return (
    <CoreMenuContext.Provider value={{ onClick: handleClick }}>
      <Space className="pluto-menu" direction="y" empty>
        {children}
      </Space>
    </CoreMenuContext.Provider>
  );
};

export interface MenuItemProps extends ButtonProps {
  itemKey: string;
}

export const MenuItem = ({
  itemKey,
  className,
  onClick,
  ...props
}: MenuItemProps): JSX.Element => {
  const { onClick: ctxOnClick } = useCoreMenuContext();
  const handleClick: ButtonProps["onClick"] = (e) => {
    ctxOnClick(itemKey);
    onClick?.(e);
  };
  return (
    <Button
      {...props}
      onClick={handleClick}
      variant="text"
      className={clsx("pluto-menu-item", className)}
    />
  );
};
