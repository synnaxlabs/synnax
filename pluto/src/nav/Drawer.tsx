// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/nav/Drawer.css";

import { type bounds, type box, location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { Errors } from "@/errors";
import { type BarProps } from "@/nav/Bar";
import { Resize } from "@/resize";
import { Eraser } from "@/vis/eraser";

export interface DrawerItem {
  key: string;
  content: ReactElement;
  sizeBounds?: Partial<bounds.Bounds>;
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
  extends
    Omit<BarProps, "onSelect" | "onResize" | "size">,
    UseDrawerReturn,
    Partial<
      Pick<
        Resize.SingleProps,
        "onResize" | "onResizeEnd" | "collapseThreshold" | "onCollapse"
      >
    > {
  eraseEnabled?: boolean;
}

export const useDrawer = ({ items, initialKey }: UseDrawerProps): UseDrawerReturn => {
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const handleSelect = (key: string): void =>
    setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return { onSelect: handleSelect, activeItem };
};

export const Drawer = ({
  activeItem,
  children,
  onSelect,
  location: loc_ = "left",
  collapseThreshold = 0.65,
  className,
  onResize,
  onResizeEnd,
  onCollapse,
  eraseEnabled,
  ...rest
}: DrawerProps): ReactElement | null => {
  const dir = location.direction(loc_);
  eraseEnabled ??= activeItem != null;
  const handleCollapse = useCallback(() => {
    if (onCollapse) onCollapse();
    else if (activeItem != null) onSelect?.(activeItem.key);
  }, [onSelect, activeItem?.key, onCollapse]);
  const { erase } = Eraser.use({ enabled: eraseEnabled });
  const handleResize = useCallback(
    (size: number, box: box.Box) => {
      erase(box);
      onResize?.(size, box);
    },
    [erase, onResize],
  );
  const handleResizeEnd = useCallback(
    (size: number, box: box.Box) => {
      erase(box);
      onResizeEnd?.(size, box);
    },
    [erase, onResizeEnd],
  );
  const { content, sizeBounds, initialSize: size = 0 } = activeItem ?? {};
  return (
    <Resize.Single
      className={CSS(
        CSS.B("nav-drawer"),
        CSS.dir(dir),
        CSS.visible(activeItem != null),
        className,
      )}
      collapseThreshold={collapseThreshold}
      onCollapse={handleCollapse}
      location={loc_}
      onResize={handleResize}
      onResizeEnd={handleResizeEnd}
      sizeBounds={sizeBounds}
      size={size}
      {...rest}
    >
      <Errors.Boundary>{content}</Errors.Boundary>
    </Resize.Single>
  );
};
