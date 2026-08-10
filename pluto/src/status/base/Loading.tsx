// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/base/Loading.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";

export interface LoadingProps extends Flex.BoxProps {}

/**
 * Loading centers an indicator in its container and holds it invisible until the
 * wait is long enough to be worth reporting, so a fast read never flashes one.
 * Defaults to the inline glyph; pass {@link Orbital} for a surface with room for
 * it. Inline surfaces (a tab chip, an icon slot) should render
 * {@link Icon.Loading} directly instead of wrapping it here.
 */
export const Loading = ({
  className,
  children,
  ...rest
}: LoadingProps): ReactElement => (
  <Flex.Box
    center
    grow
    className={CSS(CSS.BE("status", "loading"), className)}
    {...rest}
  >
    {children ?? <Icon.Loading />}
  </Flex.Box>
);
