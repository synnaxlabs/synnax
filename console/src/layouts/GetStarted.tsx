// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/GetStarted.css";

import { Logo } from "@synnaxlabs/media";
import { Button, Eraser, Flex, Icon, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Workspace } from "@/workspace";

export const GetStarted = (): ReactElement => <Overview />;

const Overview = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  const handleWorkspace = useCallback<NonNullable<Button.ButtonProps["onClick"]>>(
    () => placeLayout(Workspace.CREATE_LAYOUT),
    [placeLayout],
  );
  return (
    <Eraser.Eraser>
      <Flex.Box y full className={CSS.B("get-started")} align="center">
        <Logo variant="title" className="console-get-started__logo" />
        <Flex.Box x full="x" justify="center" gap={30} wrap>
          <Flex.Box y align="center">
            <Text.Text level="h1">Your Workspaces</Text.Text>
            <Button.Button onClick={handleWorkspace} style={{ width: "fit-content" }}>
              <Icon.Add />
              Create a Workspace
            </Button.Button>
          </Flex.Box>
          <Flex.Box y align="center">
            <Text.Text level="h1">Recent Ranges</Text.Text>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Eraser.Eraser>
  );
};
