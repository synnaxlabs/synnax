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

import { BottomNavBar } from "@/modals/BottomNavBar";

export interface ModalContentLayoutProps extends PropsWithChildren, Flex.BoxProps {
  footer: React.ReactNode;
}

export const ModalContentLayout = ({
  children,
  footer,
  style,
  ...rest
}: ModalContentLayoutProps): ReactElement => (
  <Flex.Box y grow justify="center">
    <Flex.Box
      y
      grow
      align="start"
      justify="center"
      style={{ padding: "5rem", ...style }}
      {...rest}
    >
      {children}
    </Flex.Box>
    <BottomNavBar>{footer}</BottomNavBar>
  </Flex.Box>
);
