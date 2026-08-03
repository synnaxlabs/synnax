// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, OS } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { Connection, type ConnectionCluster } from "@/platform/shell/Connection";
import { Version } from "@/platform/version";
import { Window } from "@/platform/window";
import { Session } from "@/session";

interface IslandProps extends Flex.BoxProps {
  children: ReactNode;
}

const Island = ({ children, ...rest }: IslandProps): ReactElement => (
  <Flex.Box
    x
    align="center"
    className={CSS(CSS.BE("shell", "island"), CSS.BE("shell", "frost"))}
    {...rest}
  >
    {children}
  </Flex.Box>
);

export interface NavProps {
  connection?: ConnectionCluster | null;
}

/**
 * Floating chrome for pre-workspace surfaces: islands along the top edge for
 * window controls, version, and connection state. Web builds and child windows
 * show only the connection island.
 */
export const Nav = ({ connection }: NavProps): ReactElement => {
  const os = OS.use();
  const chrome = Session.Runtime.ENGINE !== "web" && Session.Runtime.isMainWindow();
  return (
    <Flex.Box x justify="between" align="start" className={CSS.BE("shell", "islands")}>
      <Flex.Box x align="center" gap="medium">
        {chrome && os === "macOS" && (
          <Island>
            <Window.Controls visibleIfOS="macOS" forceOS={os} />
          </Island>
        )}
        {chrome && (
          <Island data-tauri-drag-region>
            <Version.Badge />
          </Island>
        )}
      </Flex.Box>
      <Flex.Box x align="center" gap="medium">
        <Connection cluster={connection} />
        {chrome && os === "Windows" && (
          <Island>
            <Window.Controls visibleIfOS="Windows" forceOS={os} />
          </Island>
        )}
      </Flex.Box>
    </Flex.Box>
  );
};
