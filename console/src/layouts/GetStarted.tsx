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
import { Icon, Logo } from "@synnaxlabs/media";
import { Align, Button, Eraser, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
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

  const handleDocs = useCallback<NonNullable<Text.LinkProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      placeLayout(Docs.LAYOUT);
    },
    [placeLayout],
  );

  return (
    <Align.Center className="console-get-started" align="center" size={6}>
      <Logo variant="title" className="console-get-started__logo" />
      <Text.Text level="h1">Get Started</Text.Text>
      <Align.Space x size="large" justify="center" wrap>
        <Button.Button
          startIcon={<Icon.Cluster />}
          onClick={handleCluster}
          size="large"
        >
          Connect a Cluster
        </Button.Button>
        <Button.Button
          startIcon={<Icon.Visualize />}
          onClick={handleVisualize}
          size="large"
        >
          Create a Visualization
        </Button.Button>
      </Align.Space>
      <Text.Link target="_blank" level="h4" onClick={handleDocs}>
        Read the Documentation
      </Text.Link>
    </Align.Center>
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
      <Align.Center
        className="console-get-started"
        size={6}
        y
        style={{ padding: "200px" }}
      >
        <Logo variant="title" className="console-get-started__logo" />
        <Align.Space x style={{ width: "100%" }} justify="center" size={30} wrap>
          <Align.Space y>
            <Text.Text level="h1">Your Workspaces</Text.Text>
            <Workspace.Recent />
            <Button.Button
              startIcon={<Icon.Add />}
              onClick={handleWorkspace}
              style={{ width: "fit-content" }}
            >
              Create a Workspace
            </Button.Button>
          </Align.Space>
          <Align.Space y align="center">
            <Text.Text level="h1">Recent Ranges</Text.Text>
          </Align.Space>
        </Align.Space>
      </Align.Center>
    </Eraser.Eraser>
  );
};
