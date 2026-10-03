// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/lyra/css";
import { type PropsWithChildren, type ReactElement } from "react";

export interface TileProps extends PropsWithChildren {
  /** status tints the tile for a page that reports an outcome. */
  status?: "success" | "error";
}

/** Tile sets an icon apart above a title. */
export const Tile = ({ status, children }: TileProps): ReactElement => (
  <span className={CSS.cls("portal-tile", status != null && `portal-tile--${status}`)}>
    {children}
  </span>
);
