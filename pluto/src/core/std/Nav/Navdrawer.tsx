// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useState } from "react";

import { CSS } from "@/core/css";
import { NavbarProps } from "@/core/std/Nav/Navbar";
import { Resize, ResizeProps } from "@/core/std/Resize";

import "@/core/std/Nav/Navdrawer.css";

import { Location } from "@synnaxlabs/x";

export interface NavDrawerItem {
  key: string;
  content: ReactElement;
  minSize?: number;
  maxSize?: number;
  initialSize?: number;
}

export interface UseNavDrawerProps {
  initialKey?: string;
  items: NavDrawerItem[];
}

export interface UseNavDrawerReturn {
  activeItem?: NavDrawerItem;
  onSelect?: (key: string) => void;
}

export interface NavDrawerProps
  extends Omit<NavbarProps, "onSelect" | "onResize">,
    UseNavDrawerReturn,
    Partial<Pick<ResizeProps, "onResize" | "collapseThreshold">> {}

export const useNavDrawer = ({
  items,
  initialKey,
}: UseNavDrawerProps): UseNavDrawerReturn => {
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const handleSelect = (key: string): void =>
    setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return { onSelect: handleSelect, activeItem };
};

export const Navdrawer = ({
  activeItem,
  children,
  onSelect,
  location = "left",
  collapseThreshold = 0.65,
  className,
  ...props
}: NavDrawerProps): ReactElement | null => {
  if (activeItem == null) return null;
  const dir = new Location(location).direction;
  const { key, content, ...rest } = activeItem;
  const handleCollapse = useCallback(() => onSelect?.(key), [onSelect, key]);
  return (
    <Resize
      className={CSS(CSS.BE("navdrawer", "content"), CSS.dir(dir), className)}
      collapseThreshold={collapseThreshold}
      onCollapse={handleCollapse}
      location={location}
      {...rest}
      {...props}
    >
      {content}
    </Resize>
  );
};
