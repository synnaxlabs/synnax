// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/pluto";
import { type MouseEvent, type ReactElement, useCallback } from "react";

import { CSS } from "@/css";

export interface ActionsProps extends Flex.BoxProps {}

export const Actions = ({
  className,
  onClick,
  ...props
}: ActionsProps): ReactElement => {
  const handleClick = useCallback(
    (e: MouseEvent<HTMLDivElement>) => {
      e.stopPropagation();
      onClick?.(e);
    },
    [onClick],
  );

  return (
    <Flex.Box
      className={CSS(CSS.BE("task-controls", "actions"), className)}
      align="center"
      x
      justify="end"
      onClick={handleClick}
      {...props}
    />
  );
};
