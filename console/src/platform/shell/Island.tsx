// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/shell/Shell.css";

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface IslandsProps extends Flex.BoxProps {}

/** The row that floats islands over the top edge of a full-window surface. */
export const Islands = ({ className, ...rest }: IslandsProps): ReactElement => (
  <Flex.Box
    x
    justify="between"
    align="start"
    className={CSS.cls(CSS.BE("shell", "islands"), className)}
    {...rest}
  />
);

export interface IslandProps extends Flex.BoxProps {}

/** A floating chip along the shell's top edge: island shape over frosted glass. */
export const Island = ({ className, ...rest }: IslandProps): ReactElement => (
  <Flex.Box
    x
    align="center"
    className={CSS.cls(CSS.BE("shell", "island"), CSS.BE("shell", "frost"), className)}
    {...rest}
  />
);
