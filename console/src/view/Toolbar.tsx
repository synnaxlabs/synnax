// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { useContext } from "@/view/context";

export interface ToolbarProps extends PropsWithChildren {}

export const Toolbar = ({ children }: ToolbarProps): ReactElement | null => {
  const { editable } = useContext("View.Toolbar");
  if (!editable) return null;
  return (
    <Flex.Box
      x
      bordered
      style={{ padding: "1.5rem" }}
      background={1}
      justify="between"
      align="center"
    >
      {children}
    </Flex.Box>
  );
};
