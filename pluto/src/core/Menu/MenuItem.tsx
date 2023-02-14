// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";
import { FunctionComponent } from "react";
import { Button, ButtonIconProps, ButtonProps } from "../Button";
import { useMenuContext } from "./Menu";

import "./MenuItem.css";

const menuItemFactory =
  <E extends Pick<ButtonProps, "className" | "onClick">>(
    Base: FunctionComponent<E>
  ): FunctionComponent<E & { itemKey: string }> =>
  // eslint-disable-next-line react/display-name
  ({ itemKey, className, onClick, ...props }): JSX.Element => {
    const { onClick: ctxOnClick, selected } = useMenuContext();
    const handleClick: ButtonProps["onClick"] = (e) => {
      ctxOnClick(itemKey);
      onClick?.(e);
    };
    const _selected = selected === itemKey;
    return (
      // @ts-expect-error
      <Base
        {...props}
        onClick={handleClick}
        variant="text"
        className={clsx(
          "pluto-menu-item",
          _selected && "pluto-menu-item--selected",
          className
        )}
      />
    );
  };

export interface MenuItemProps extends ButtonProps {
  itemKey: string;
}
export const MenuItem = menuItemFactory(Button);

export interface MenuItemIconProps extends ButtonIconProps {
  itemKey: string;
}
export const MenuItemIcon = menuItemFactory(Button.Icon);
