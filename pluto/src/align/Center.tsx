// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/align/Center.css";

import { type ReactElement } from "react";

import { Space, type SpaceElementType, type SpaceProps } from "@/align/Space";
import { CSS } from "@/css";

/**
 * An element whose width and height is 100% and whose alignment and justification
 * is centered. Props are the same as {@link Space}.
 */
export const Center = <E extends SpaceElementType = "div">({
  className,
  justify = "center",
  align = "center",
  ...rest
}: SpaceProps<E>): ReactElement => (
  // @ts-expect-error - generic element issues
  <Space<E>
    justify={justify}
    align={align}
    className={CSS(CSS.BM("space", "centered"), className)}
    {...rest}
  />
);
