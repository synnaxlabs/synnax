// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useState } from "react";

import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { Resize } from "@/resize";

export interface DrawerProps extends Resize.SingleProps {
  collapsed?: boolean;
  collapseThreshold?: number;
  onCollapse?: () => void;
}

export const Drawer = ({
  onCollapse,
  collapsed = false,
  collapseThreshold = 65,
  onResize,
  onResizeEnd,
  className,
  ...rest
}: DrawerProps) => {
  const collapsedRef = useSyncedRef(collapsed);
  const [dragCollapsed, setDragCollapsed] = useState(false);
  const underCollapseThreshold = useCallback(
    (size: number, { dragSize }: Resize.HandlerExtra) =>
      size - dragSize > collapseThreshold,
    [collapseThreshold],
  );
  const handleResize = useCallback(
    (size: number, extra: Resize.HandlerExtra) => {
      if (collapsedRef.current) return;
      const next = underCollapseThreshold(size, extra);
      setDragCollapsed(next);
      if (!next) onResize?.(size, extra);
    },
    [onResize, underCollapseThreshold],
  );
  const handleResizeEnd = useCallback(
    (size: number, extra: Resize.HandlerExtra) => {
      setDragCollapsed(false);
      if (collapsedRef.current) return;
      if (underCollapseThreshold(size, extra)) onCollapse?.();
      else onResizeEnd?.(size, extra);
    },
    [onResizeEnd, onCollapse, underCollapseThreshold],
  );
  return (
    <Resize.Single
      className={CSS(CSS.visible(!(collapsed || dragCollapsed)), className)}
      onResize={handleResize}
      onResizeEnd={handleResizeEnd}
      {...rest}
    />
  );
};
