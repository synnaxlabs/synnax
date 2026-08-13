// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/shell/Shell.css";

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { type ConnectionCluster } from "@/platform/shell/Connection";
import { Nav } from "@/platform/shell/Nav";
import { Nebula } from "@/platform/shell/Nebula";

export interface FrameProps {
  className?: string;
  /** Target Core shown in the connection island; omit to hide it. */
  connection?: ConnectionCluster | null;
  children: ReactNode;
}

/**
 * Full-window shell for pre-workspace surfaces: window chrome, halftone nebula,
 * and the centered card that hosts the surface's content.
 */
export const Frame = ({
  className,
  connection,
  children,
}: FrameProps): ReactElement => (
  <Flex.Box y full empty className={CSS(CSS.B("shell"), className)}>
    <Nav connection={connection} />
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
      <Nebula />
      <Flex.Box className={CSS.BE("shell", "stage")} grow={false}>
        <Flex.Box
          y
          empty
          className={CSS(CSS.BE("shell", "card"), CSS.BE("shell", "frost"))}
          grow={false}
        >
          {children}
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
