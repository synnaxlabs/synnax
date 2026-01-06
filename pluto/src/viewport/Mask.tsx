// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/viewport/Mask.css";

import { box } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { CSS } from "@/css";
import { type Mode, type UseReturn } from "@/viewport/use";

type DivProps = React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>;

export interface MaskProps
  extends
    Omit<UseReturn, "ref">,
    Omit<DivProps, "onDragStart" | "onDragEnd" | "onDrag" | "ref" | "onDoubleClick"> {}

const MODE_CURSORS: Record<Mode, CSSProperties["cursor"]> = {
  select: "pointer",
  zoom: "crosshair",
  pan: "grab",
  zoomReset: "crosshair",
  click: "pointer",
  cancel: "default",
};

export const Mask = ({
  className,
  mode,
  maskBox,
  children,
  style,
  ...rest
}: MaskProps): ReactElement | null => (
  <div
    className={CSS(CSS.noSelect, CSS.BE("viewport-mask", "container"), className)}
    style={{
      cursor: MODE_CURSORS[mode],
      ...style,
    }}
    {...rest}
  >
    <div
      style={{
        ...box.css(maskBox),
        display: box.areaIsZero(maskBox) ? "none" : "block",
      }}
      className={CSS.BE("viewport-mask", "selection")}
    />
    {children}
  </div>
);
