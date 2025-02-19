// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { BottomNavBar } from "@/modals/BottomNavBar";

export interface ModalContentLayoutProps extends PropsWithChildren {
  footer: React.ReactNode;
}

export const ModalContentLayout = ({
  children,
  footer,
}: ModalContentLayoutProps): ReactElement => (
  <Align.Space direction="y" grow justify="center">
    <Align.Space
      direction="y"
      grow
      align="start"
      justify="center"
      style={{ padding: "5rem" }}
    >
      {children}
    </Align.Space>
    <BottomNavBar>{footer}</BottomNavBar>
  </Align.Space>
);
