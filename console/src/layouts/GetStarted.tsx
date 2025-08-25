// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/GetStarted.css";

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Logo } from "@synnaxlabs/media";
import { Button, Eraser, Flex, Icon, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { Vis } from "@/vis";
import { Workspace } from "@/workspace";

export const GetStarted = (): ReactElement => {
  const client = Synnax.use();
  return client == null ? <NoCluster /> : <Overview />;
};

const NoCluster = (): ReactElement => {
  const windowKey = useSelectWindowKey() as string;
  const placeLayout = Layout.usePlacer();
  const dispatch = useDispatch();

  // As a note, we need to stop propagation on these events so that we don't
  // trigger the 'onSelect' handler of the tab we're in. This means we appropriately
  // select the new layout when we create it.
  const handleCluster = useCallback<NonNullable<Button.ButtonProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      placeLayout(Cluster.CONNECT_LAYOUT);
    },
    [placeLayout],
  );

  const handleVisualize = useCallback<NonNullable<Button.ButtonProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      placeLayout(Vis.createSelectorLayout());
      dispatch(
        Layout.setNavDrawerVisible({ windowKey, key: Vis.TOOLBAR.key, value: true }),
      );
    },
    [placeLayout, dispatch, windowKey],
  );

  const handleDocs = useCallback<NonNullable<Text.TextProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      placeLayout(Docs.LAYOUT);
    },
    [placeLayout],
  );

  return (
    <Flex.Box className={CSS.B("get-started")} gap={4} full align="center">
      <Logo variant="title" className="console-get-started__logo" />
      <Text.Text level="h1">Get Started</Text.Text>
      <Flex.Box x gap="large" justify="center" wrap>
        <Button.Button onClick={handleCluster} size="large" variant="filled">
          <Icon.Cluster />
          Connect a Cluster
        </Button.Button>
        <Button.Button onClick={handleVisualize} size="large" variant="filled">
          <Icon.Visualize />
          Create a Visualization
        </Button.Button>
      </Flex.Box>
      <Text.Text variant="link" target="_blank" level="h4" onClick={handleDocs}>
        Read the Documentation
      </Text.Text>
    </Flex.Box>
  );
};

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
