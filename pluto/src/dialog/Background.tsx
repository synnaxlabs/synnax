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
import { Flex } from "@/flex";

export interface BackgroundProps extends Flex.BoxProps {
  visible: boolean;
}

export const BACKGROUND_CLASS = CSS.BE("dialog", "bg");

export const Background = ({
  children,
  visible,
  ...rest
}: BackgroundProps): ReactElement => (
  <Flex.Box
    className={CSS(BACKGROUND_CLASS, CSS.visible(visible))}
    empty
    align="center"
    {...rest}
  >
    {children}
  </Flex.Box>
);
