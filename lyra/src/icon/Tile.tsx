// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/icon/Tile.css";

import { type ReactElement } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";

/** Props for {@link Tile}. */
export interface TileProps {
  /** The icon to frame. */
  children: ReactElement;
  /** The side of the square. Defaults to "large". */
  size?: Component.Size;
  className?: string;
}

/**
 * An icon framed in a square, so that icons of different shapes in one list line up
 * at one size. The frame is decorative and hidden from assistive technology.
 */
export const Tile = ({
  children,
  size = "large",
  className,
}: TileProps): ReactElement => (
  <span
    aria-hidden
    className={CSS.cls(CSS.B("icon-tile"), CSS.BM("icon-tile", size), className)}
  >
    {children}
  </span>
);
