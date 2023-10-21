// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useState } from "react";

import { type box, location } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { type BarProps } from "@/nav/Bar";
import { Resize } from "@/resize";
import { Eraser } from "@/vis/eraser";

import "@/nav/Drawer.css";

export interface DrawerItem {
  key: string;
  content: ReactElement;
  minSize?: number;
  maxSize?: number;
  initialSize?: number;
}

export interface UseDrawerProps {
  initialKey?: string;
  items: DrawerItem[];
}

export interface UseDrawerReturn {
  activeItem?: DrawerItem;
  onSelect?: (key: string) => void;
}

export interface DrawerProps
  extends Omit<BarProps, "onSelect" | "onResize">,
    UseDrawerReturn,
    Partial<Pick<Resize.SingleProps, "onResize" | "collapseThreshold">> {}

export const useDrawer = ({ items, initialKey }: UseDrawerProps): UseDrawerReturn => {
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const handleSelect = (key: string): void =>
    setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return { onSelect: handleSelect, activeItem };
};

export const Drawer = Aether.wrap<DrawerProps>(
  "Nav.Drawer",
  ({
    aetherKey,
    activeItem,
    children,
    onSelect,
    location: loc_ = "left",
    collapseThreshold = 0.65,
    className,
    onResize,
    ...props
  }): ReactElement | null => {
    if (activeItem == null) return null;
    const dir = location.direction(loc_);
    const { key, content, ...rest } = activeItem;
    const handleCollapse = useCallback(() => onSelect?.(key), [onSelect, key]);
    const erase = Eraser.use({ aetherKey });
    const handleResize = useCallback(
      (size: number, box: box.Box) => {
        onResize?.(size, box);
        erase(box);
      },
      [onResize, erase],
    );
    return (
      <Resize.Single
        className={CSS(CSS.BE("navdrawer", "content"), CSS.dir(dir), className)}
        collapseThreshold={collapseThreshold}
        onCollapse={handleCollapse}
        location={loc_}
        onResize={handleResize}
        {...rest}
        {...props}
      >
        {content}
      </Resize.Single>
    );
  },
);
