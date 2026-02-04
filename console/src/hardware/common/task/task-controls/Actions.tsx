// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";

export interface ActionsProps extends PropsWithChildren, Omit<Flex.BoxProps, "children"> {}

export const Actions = ({
  children,
  className,
  ...props
}: ActionsProps): ReactElement => (
  <Flex.Box
    className={CSS(CSS.BE("task-controls", "actions"), className)}
    align="center"
    x
    justify="end"
    {...props}
  >
    {children}
  </Flex.Box>
);
