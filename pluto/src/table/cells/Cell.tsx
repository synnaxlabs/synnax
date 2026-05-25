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
}

// Cell is the primitive <td> wrapper used by variant cell components in
// @/table/cells. The connected Table renders cell content through the
// variant registry; variant components compose this primitive to get
// consistent styling and selection visuals.
export const Cell = ({
  ref,
  children,
  className,
  selected = false,
  ...rest
}: CellProps): ReactElement => (
  <td
    ref={ref}
    {...rest}
    className={CSS(CSS.BE("table", "cell"), CSS.selected(selected), className)}
  >
    {children}
  </td>
);
