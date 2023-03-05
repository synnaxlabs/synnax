// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FunctionComponent } from "react";

import { Button, ButtonIconProps, ButtonProps } from "../Button";

import { useMenuContext } from "./Menu";

import { CSS } from "@/css";

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
        className={CSS(CSS.B("menu-item"), CSS.selected(_selected), className)}
      />
    );
  };

export interface MenuItemProps extends ButtonProps {
  itemKey: string;
}
export const CoreMenuItem = menuItemFactory(Button);

export interface MenuItemIconProps extends ButtonIconProps {
  itemKey: string;
}
const MenuItemIcon = menuItemFactory(Button.Icon);

const MenuItemLink = menuItemFactory(Button.Link);
export interface MenuItemLinkProps extends ButtonProps {
  itemKey: string;
}

type CoreMenuItemType = typeof CoreMenuItem;

export interface MenuItemType extends CoreMenuItemType {
  Icon: typeof MenuItemIcon;
  Link: typeof MenuItemLink;
}

export const MenuItem = CoreMenuItem as MenuItemType;
MenuItem.Icon = MenuItemIcon;
MenuItem.Link = MenuItemLink;
