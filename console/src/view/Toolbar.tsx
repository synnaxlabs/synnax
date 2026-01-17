// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { useContext } from "@/view/context";

export interface ToolbarProps extends Flex.BoxProps {}

export const Toolbar = ({ className, ...rest }: ToolbarProps): ReactElement => {
  const { editable } = useContext("View.Toolbar");
  if (!editable) return <Divider.Divider x />;
  return (
    <Flex.Box
      x
      bordered
      background={1}
      justify="between"
      align="center"
      className={CSS(CSS.BE("view", "toolbar"), className)}
      {...rest}
    />
  );
};
