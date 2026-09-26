// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

/** Props for {@link Actions}. */
export interface ActionsProps extends Omit<Flex.BoxProps, "children" | "direction"> {
  children?: ReactNode;
}

/** The trailing slot of a {@link Header}, holding its buttons. */
export const Actions = ({ children = [], ...rest }: ActionsProps): ReactElement => (
  <Flex.Box
    x
    gap="small"
    align="center"
    className={CSS.BE("header", "actions")}
    {...rest}
  >
    {children}
  </Flex.Box>
);
