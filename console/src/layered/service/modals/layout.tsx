// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, type Icon } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";
import { BottomNavBar } from "@/layered/service/modals/BottomNavBar";
import { Header } from "@/layered/service/modals/Header";

export interface ModalContentLayoutProps extends PropsWithChildren, Flex.BoxProps {
  /** The dotted title rendered in the header breadcrumb. Omit to render no header. */
  title?: string;
  /** The leading header icon. */
  icon?: Icon.ReactElement | string;
  /** The footer content rendered in the bottom nav bar. Omit to render no footer. */
  footer?: React.ReactNode;
}

export const ModalContentLayout = ({
  children,
  title,
  icon,
  footer,
  className,
  ...rest
}: ModalContentLayoutProps): ReactElement => (
  <Flex.Box y grow empty>
    {title != null && <Header name={title} icon={icon} />}
    <Flex.Box
      y
      grow
      justify="center"
      className={CSS(CSS.B("modal-content-layout"), className)}
      {...rest}
    >
      {children}
    </Flex.Box>
    {footer != null && <BottomNavBar>{footer}</BottomNavBar>}
  </Flex.Box>
);
