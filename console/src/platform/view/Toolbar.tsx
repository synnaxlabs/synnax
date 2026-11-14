// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface ToolbarProps extends Flex.BoxProps {}

export const Toolbar = ({ className, ...rest }: ToolbarProps): ReactElement => (
  <Flex.Box
    x
    background={1}
    justify="between"
    align="center"
    className={CSS.cls(CSS.BE("view", "toolbar"), className)}
    {...rest}
  />
);
