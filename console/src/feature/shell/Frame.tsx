// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/shell/Shell.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Nebula } from "@synnaxlabs/lyra/nebula";
import { type ReactElement, type ReactNode } from "react";

import { Nav, type NavProps } from "@/feature/shell/Nav";
import { CSS } from "@/platform/css";

export interface FrameProps extends NavProps {
  className?: string;
  children: ReactNode;
}

/**
 * Full-window shell for pre-workspace surfaces: window chrome, halftone nebula,
 * and the centered card that hosts the surface's content.
 */
export const Frame = ({ className, children, ...rest }: FrameProps): ReactElement => (
  <Flex.Box y full empty className={CSS.cls(CSS.B("shell"), className)}>
    <Nav {...rest} />
    <Flex.Box
      y
      align="center"
      justify="center"
      background={1}
      gap="huge"
      grow
      data-tauri-drag-region
      className={CSS.BE("shell", "content")}
    >
      <Nebula.Nebula />
      <Flex.Box className={CSS.BE("shell", "stage")} grow={false}>
        <Flex.Box
          y
          empty
          className={CSS.cls(CSS.BE("shell", "card"), CSS.BE("shell", "frost"))}
          grow={false}
        >
          {children}
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);

export interface PageProps extends FrameProps {}

/** Full-window shell whose content fills the window below the nav. */
export const Page = ({ className, children, ...rest }: PageProps): ReactElement => (
  <Flex.Box y full empty className={CSS.cls(CSS.B("shell"), className)}>
    <Nav {...rest} />
    <Flex.Box
      y
      grow
      empty
      background={1}
      data-tauri-drag-region
      className={CSS.BE("shell", "page")}
    >
      {children}
    </Flex.Box>
  </Flex.Box>
);
