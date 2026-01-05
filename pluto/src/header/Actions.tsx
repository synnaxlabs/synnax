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

export interface ActionsProps extends Omit<Flex.BoxProps, "children" | "direction"> {
  children?: ReactNode;
}

/**
 * Custom actions to render on the right side of the header.
 *
 * @param children - The actions to render. If the action is of type
 * {@link ButtonIconProps}, a correectly sized {@link ButtonIconOnly} is rendered
 * using the given props. If the action is a JSX element, it is renderered directly.
 * It's a good idea to prefer the latter in almost all cases for simplicity.
 */
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
