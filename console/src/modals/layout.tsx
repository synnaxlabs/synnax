// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modals/layout.css";

import { Flex } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";
import { BottomNavBar } from "@/modals/BottomNavBar";

export interface ModalContentLayoutProps extends PropsWithChildren, Flex.BoxProps {
  footer: React.ReactNode;
}

export const ModalContentLayout = ({
  children,
  footer,
  className,
  ...rest
}: ModalContentLayoutProps): ReactElement => (
  <Flex.Box y grow justify="center">
    <Flex.Box
      y
      grow
      justify="center"
      className={CSS(CSS.B("modal-content-layout"), className)}
      {...rest}
    >
      {children}
    </Flex.Box>
    <BottomNavBar>{footer}</BottomNavBar>
  </Flex.Box>
);
