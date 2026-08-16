// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, OS } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Shell } from "@/platform/shell";
import { Version } from "@/platform/version";
import { Window } from "@/platform/window";
import { Session } from "@/session";

export interface NavProps {
  connection?: Shell.ConnectionCluster | null;
}

/**
 * Floating chrome for pre-workspace surfaces: islands along the top edge for
 * window controls, version, and connection state. Web builds and child windows
 * show only the connection island.
 */
export const Nav = ({ connection }: NavProps): ReactElement => {
  const os = OS.use();
  const chrome = Session.Runtime.ENGINE === "tauri" && Session.Runtime.isMainWindow();
  return (
    <Shell.Islands>
      <Flex.Box x align="center" gap="medium">
        {chrome && os === "macOS" && (
          <Shell.Island>
            <Window.Controls visibleIfOS="macOS" forceOS={os} />
          </Shell.Island>
        )}
        {chrome && (
          <Shell.Island data-tauri-drag-region>
            <Version.Badge />
          </Shell.Island>
        )}
      </Flex.Box>
      <Flex.Box x align="center" gap="medium">
        <Shell.Connection cluster={connection} />
        {chrome && os === "Windows" && (
          <Shell.Island>
            <Window.Controls visibleIfOS="Windows" forceOS={os} />
          </Shell.Island>
        )}
      </Flex.Box>
    </Shell.Islands>
  );
};
