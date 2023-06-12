// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ForwardedRef, ReactElement, forwardRef } from "react";

import { Space, SpaceElementType, SpaceProps } from "./Space";

import { CSS } from "@/core/css";

import "@/core/std/Space/SpaceCentered.css";

export const CoreSpaceCentered = <E extends SpaceElementType = "div">(
  { className, justify = "center", align = "center", ...props }: SpaceProps<E>,
  ref: ForwardedRef<JSX.IntrinsicElements[E]>
): ReactElement => (
  // @ts-expect-error
  <Space
    ref={ref}
    justify={justify}
    align={align}
    className={CSS(CSS.BM("space", "centered"), className)}
    {...props}
  />
);

export const SpaceCentered = forwardRef(CoreSpaceCentered) as <
  E extends SpaceElementType = "div"
>(
  props: SpaceProps<E>
) => ReactElement;
