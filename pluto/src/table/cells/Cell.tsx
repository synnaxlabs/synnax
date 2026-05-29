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
  // height is the row's stored size in px. The inner content div is locked
  // to this so a cell whose content wants more vertical room (e.g. wrapped
  // text) clips at the row's bottom instead of growing the row.
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
  ...rest
}: CellProps): ReactElement => (
  <td
    ref={ref}
    {...rest}
    className={CSS(CSS.BE("table", "cell"), CSS.selected(selected), className)}
  >
    <div className={CSS.BEM("table", "cell", "content")} style={{ height }}>
      {children}
    </div>
  </td>
);
