// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";
import { createContext, PropsWithChildren, useContext } from "react";
import { Button, ButtonProps } from "../Button";

interface CoreMenuContextProps {
  onClick: (key: string) => void;
}

export const CoreMenuContext = createContext<CoreMenuContextProps>({ onClick: () => {} });

export interface MenuProps extends PropsWithChildren {
  onClick?: (key: string) => void;
}

const useCoreMenuContext = () => useContext(CoreMenuContext);

export const Menu = ({ children, onClick }: MenuProps): JSX.Element => {
  const handleClick = (key: string) => {
    onClick?.(key);
  };
  return (
    <CoreMenuContext.Provider value={{ onClick: handleClick }}>
        <div className="pluto-menu">
          {children}
        </div>
    </CoreMenuContext.Provider>
  );
}

export interface MenuItemProps extends Omit<ButtonProps, 'onClick'> {
  itemKey: string;
}

export const MenuItem = ({ itemKey, className, ...props }: MenuItemProps): JSX.Element => {
  const { onClick } = useCoreMenuContext()
  return (
    <Button 
      {...props} 
      onClick={() => onClick?.(itemKey)} 
      variant="text" 
      className={clsx('pluto-menu-item', className)}
    />
  );
} 
