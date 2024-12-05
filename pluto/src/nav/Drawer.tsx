// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/nav/Drawer.css";

import { type box, location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useLayoutEffect, useState } from "react";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { type BarProps } from "@/nav/Bar";
import { Resize } from "@/resize";
import { Eraser } from "@/vis/eraser";

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
    const dir = location.direction(loc_);
    const handleCollapse = useCallback(
      () => activeItem != null && onSelect?.(activeItem.key),
      [onSelect, activeItem?.key],
    );
    const { erase, setEnabled } = Eraser.use({ aetherKey });
    const handleResize = useCallback(
      (size: number, box: box.Box) => {
        onResize?.(size, box);
        erase(box);
      },
      [onResize, erase],
    );
    useLayoutEffect(() => {
      setEnabled(activeItem != null);
    }, [activeItem, setEnabled]);
    if (activeItem == null) return null;
    const { content, key, minSize, maxSize, initialSize } = activeItem;
    return (
      <Resize.Single
        key={key}
        className={CSS(CSS.BE("navdrawer", "content"), CSS.dir(dir), className)}
        collapseThreshold={collapseThreshold}
        onCollapse={handleCollapse}
        location={loc_}
        onResize={handleResize}
        minSize={minSize}
        maxSize={maxSize}
        initialSize={initialSize}
        {...props}
      >
        {content}
      </Resize.Single>
    );
  },
);
