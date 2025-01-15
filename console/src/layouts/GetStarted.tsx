// Copyright 2024 Synnax Labs, Inc.
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
import { Align, Eraser, Synnax } from "@synnaxlabs/pluto";
import { Button } from "@synnaxlabs/pluto/button";
import { Text } from "@synnaxlabs/pluto/text";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { createSelector } from "@/layouts/Selector";
import { Vis } from "@/vis";
import { Workspace } from "@/workspace";

export const GetStarted = (): ReactElement => {
  const client = Synnax.use();
  if (client == null) return <NoCluster />;
  return <Overview />;
};

const NoCluster = (): ReactElement => {
  const windowKey = useSelectWindowKey() as string;
  const place = Layout.usePlacer();
  const dispatch = useDispatch();

  // As a note, we need to stop propagation on these events so that we don't
  // trigger the 'onSelect' handler of the tab we're in. This means we appropriately
  // select the new layout when we create it.
  const handleCluster: Button.ButtonProps["onClick"] = (e) => {
    e.stopPropagation();
    place(Cluster.connectWindowLayout);
  };

  const handleVisualize: Button.ButtonProps["onClick"] = (e) => {
    e.stopPropagation();
    place(createSelector({}));
    dispatch(
      Layout.setNavDrawerVisible({ windowKey, key: Vis.Toolbar.key, value: true }),
    );
  };

  const handleDocs: Text.LinkProps["onClick"] = (e) => {
    e.stopPropagation();
    place(Docs.createLayout());
  };

  return (
    <Align.Center className="console-get-started" align="center" size={6}>
      <Logo variant="title" className="console-get-started__logo" />
      <Text.Text level="h1">Get Started</Text.Text>
      <Align.Space direction="x" size="large" justify="center" wrap>
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
  const place = Layout.usePlacer();
  const handleWorkspace: Button.ButtonProps["onClick"] = () =>
    place(Workspace.CREATE_WINDOW_LAYOUT);

  return (
    <Eraser.Eraser>
      <Align.Center
        className="console-get-started"
        size={6}
        direction="y"
        style={{ padding: "200px" }}
      >
        <Logo variant="title" className="console-get-started__logo" />
        <Align.Space
          direction="x"
          style={{ width: "100%" }}
          justify="center"
          size={30}
          wrap
        >
          <Align.Space direction="y">
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
          <Align.Space direction="y" align="center">
            <Text.Text level="h1">Recent Ranges</Text.Text>
          </Align.Space>
        </Align.Space>
      </Align.Center>
    </Eraser.Eraser>
  );
};
