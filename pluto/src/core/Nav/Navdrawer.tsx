// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import { clamp } from "@synnaxlabs/x";
import clsx from "clsx";

import { NavbarProps, useNavbar } from "./Navbar";

import { Resize, ResizeProps } from "@/core/Resize";

import { NavMenuItem } from "./NavMenu";

import "./Navdrawer.css";

export interface NavDrawerContent {
  key: string;
  content: ReactElement;
  minSize?: number;
  maxSize?: number;
  initialSize?: number;
}

export interface NavDrawerItem extends NavDrawerContent, NavMenuItem {}

export interface UseNavDrawerProps {
  initialKey?: string;
  items: NavDrawerItem[];
}

export interface UseNavDrawerReturn {
  activeItem?: NavDrawerContent;
  menuItems?: NavMenuItem[];
  onSelect?: (key: string) => void;
}

export interface NavDrawerProps
  extends Omit<NavbarProps, "onSelect" | "onResize">,
    UseNavDrawerReturn,
    Partial<Pick<ResizeProps, "onResize">> {}

export const useNavDrawer = ({
  items,
  initialKey,
}: UseNavDrawerProps): UseNavDrawerReturn => {
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const handleSelect = (key: string): void =>
    setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return { onSelect: handleSelect, activeItem, menuItems: items };
};

export const Navdrawer = ({
  activeItem,
  menuItems = [],
  children,
  onSelect,
  onResize,
  ...props
}: NavDrawerProps): JSX.Element | null => {
  const { direction } = useNavbar(props);
  if (activeItem == null) return null;
  const { content, maxSize, minSize } = activeItem;
  let { initialSize } = activeItem;
  if (initialSize != null) initialSize = clamp(initialSize, minSize, maxSize);
  return (
    <Resize
      className={clsx(
        "pluto-navdrawer__content",
        `pluto-navdrawer__content--${direction}`,
        `pluto-navdrawer__content--${props.location}`
      )}
      onResize={onResize}
      minSize={minSize}
      maxSize={maxSize}
      initialSize={initialSize}
      {...props}
    >
      {content}
    </Resize>
  );
};
