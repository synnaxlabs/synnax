// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/task-controls/Controls.css";

import { Flex } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";

export interface FrameProps extends PropsWithChildren, Flex.BoxProps {
  /** Whether the controls are in expanded state */
  expanded?: boolean;
  /** Whether the controls are in hovered preview state */
  hovered?: boolean;
}

export const Frame = ({
  expanded = false,
  hovered = false,
  children,
  className,
  ...props
}: FrameProps): ReactElement => (
  <Flex.Box
    className={CSS(
      CSS.B("task-controls"),
      expanded && CSS.BM("task-controls", "expanded"),
      hovered && !expanded && CSS.BM("task-controls", "hovered"),
      className,
    )}
    x
    justify="between"
    empty
    bordered
    {...props}
  >
    {children}
  </Flex.Box>
);
