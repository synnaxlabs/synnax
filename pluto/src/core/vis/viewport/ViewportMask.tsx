// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSSProperties, ReactElement, forwardRef } from "react";

import { CSS } from "@/core/css";
import { UseViewportReturn, ViewportMode } from "@/core/vis/viewport/useViewport";

import "@/core/vis/viewport/ViewportMask.css";

type DivProps = React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>;

export interface ViewportMaskProps
  extends Omit<UseViewportReturn, "ref">,
    Omit<DivProps, "onDragStart" | "onDragEnd" | "onDrag" | "ref" | "onDoubleClick"> {}

const MODE_CURSORS: Record<ViewportMode, CSSProperties["cursor"]> = {
  select: "pointer",
  zoom: "crosshair",
  pan: "grab",
  zoomReset: "crosshair",
  hover: "pointer",
};

export const ViewportMask = forwardRef<HTMLDivElement, ViewportMaskProps>(
  (
    { className, mode, maskBox, children, style, ...props },
    ref
  ): ReactElement | null => (
    <div
      ref={ref}
      className={CSS(CSS.noSelect, CSS.BE("viewport-mask", "container"), className)}
      style={{
        cursor: MODE_CURSORS[mode],
        ...style,
      }}
      {...props}
    >
      <div style={maskBox.css} className={CSS.BE("viewport-mask", "selection")} />
      {children}
    </div>
  )
);
ViewportMask.displayName = "ZoomPanMask";
