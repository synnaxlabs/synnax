// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/controls/Controls.css";

import { Flex } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";

export interface FrameProps extends PropsWithChildren, Flex.BoxProps {
  /** Whether the controls are in expanded state */
  expanded?: boolean;
  /** Callback when the frame is clicked to contract (only when expanded) */
  onContract?: () => void;
}

export const Frame = ({
  expanded = false,
  onContract,
  children,
  className,
  ...props
}: FrameProps): ReactElement => (
  <Flex.Box
    className={CSS(
      CSS.B("task-controls"),
      expanded && CSS.BM("task-controls", "expanded"),
      className,
    )}
    direction={expanded ? "y" : "x"}
    justify="between"
    align="stretch"
    empty
    bordered
    pack
    rounded={1}
    borderColor={6}
    background={1}
    onClick={expanded ? onContract : undefined}
    {...props}
  >
    {children}
  </Flex.Box>
);
