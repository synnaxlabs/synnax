// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon, Logo } from "@synnaxlabs/media";
import { Text, Align, Button } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { ClusterToolbar, connectClusterWindowLayout } from "@/cluster";
import { createDocsLayout } from "@/docs";
import { useLayoutPlacer } from "@/layout/hooks";
import { setNavdrawerVisible } from "@/layout/store";
import { VisToolbar, createVis } from "@/vis";

import "@/layout/components/GetStarted.css";

export const GetStarted = (): ReactElement => {
  const placer = useLayoutPlacer();
  const dispatch = useDispatch();

  // As a note, we need to stop propagation on these events so that we don't
  // trigger the 'onSelect' handler of the tab we're in. This means we appropartiately
  // select the new layout when we create it.

  const handleCluster: Button.ButtonProps["onClick"] = (e) => {
    e.stopPropagation();
    placer(connectClusterWindowLayout);
    dispatch(setNavdrawerVisible({ key: ClusterToolbar.key, value: true }));
  };

  const handleVisualize: Button.ButtonProps["onClick"] = (e) => {
    e.stopPropagation();
    placer(createVis({}));
    dispatch(setNavdrawerVisible({ key: VisToolbar.key, value: true }));
  };

  const handleDocs: Text.LinkProps["onClick"] = (e) => {
    e.stopPropagation();
    placer(createDocsLayout());
  };

  return (
    <Align.Center className="delta-get-started" align="center" size={6}>
      <Logo variant="title" className="delta-get-started__logo" />
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
          startIcon={<Icon.Control />}
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
