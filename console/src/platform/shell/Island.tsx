// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface IslandProps extends Flex.BoxProps {}

/** A floating chip along the shell's top edge: island shape over frosted glass. */
export const Island = ({ className, ...rest }: IslandProps): ReactElement => (
  <Flex.Box
    x
    align="center"
    className={CSS(CSS.BE("shell", "island"), CSS.BE("shell", "frost"), className)}
    {...rest}
  />
);
