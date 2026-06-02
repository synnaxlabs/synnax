// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";

export interface CellProps extends React.ComponentPropsWithRef<"td"> {
  selected?: boolean;
  // height is the row's stored size in px, applied to the <td> with
  // box-sizing: border-box so the bottom border is absorbed rather than
  // added. Otherwise rows render 1px taller than rowYCursor advances and the
  // canvas value drifts upward down the table.
  height: number;
}

// Cell is the primitive <td> wrapper that gives cell variants consistent
// styling and selection visuals.
export const Cell = ({
  ref,
  children,
  className,
  selected = false,
  height,
  style,
  ...rest
}: CellProps): ReactElement => (
  <td
    ref={ref}
    {...rest}
    style={{ ...style, height }}
    className={CSS(CSS.BE("table", "cell"), CSS.selected(selected), className)}
  >
    <div className={CSS.BEM("table", "cell", "content")}>{children}</div>
  </td>
);
