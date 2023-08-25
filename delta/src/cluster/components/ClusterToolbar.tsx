// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align, Header, List, Text, componentRenderProp } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { connectClusterWindowLayout } from "@/cluster/components/ConnectCluster";
import { RenderableCluster } from "@/cluster/core";
import { setActiveCluster, useSelectCluster, useSelectClusters } from "@/cluster/store";
import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { useLayoutPlacer, NavDrawerItem } from "@/layout";

import "@/cluster/components/ClusterToolbar.css";

const Content = (): ReactElement => {
  const dispatch = useDispatch();
  const data = Object.values(useSelectClusters());
  const active = useSelectCluster();
  const openWindow = useLayoutPlacer();

  const selected = active != null ? [active?.key] : [];

  const actions = [
    {
      children: <Icon.Add />,
      onClick: () => openWindow(connectClusterWindowLayout),
    },
  ];

  const handleSelect = ([key]: string[]): void => {
    dispatch(setActiveCluster(key ?? null));
  };

  return (
    <Align.Space empty>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Cluster />}>Clusters</ToolbarTitle>
        <Header.Actions>{actions}</Header.Actions>
      </ToolbarHeader>
      <List.List<string, RenderableCluster> data={data}>
        <List.Selector value={selected} onChange={handleSelect} allowMultiple={false} />
        <List.Core.Virtual itemHeight={30}>
          {componentRenderProp(ListItem)}
        </List.Core.Virtual>
      </List.List>
    </Align.Space>
  );
};

const ListItem = ({
  entry: { key, name },
  selected,
  onSelect,
  style,
}: List.ItemProps<string, RenderableCluster>): ReactElement => (
  <Align.Space
    direction="x"
    align="center"
    justify="spaceBetween"
    onDoubleClick={() => onSelect?.(key)}
    className={CSS(
      CSS.BE("cluster-toolbar-list", "item"),
      selected && CSS.M("selected")
    )}
    style={style}
  >
    <Text.Text level="p">{name}</Text.Text>
  </Align.Space>
);

/** Configuration and content for the cluster nav drawer toolbar. */
export const ClusterToolbar: NavDrawerItem = {
  key: "clusters",
  content: <Content />,
  icon: <Icon.Cluster />,
  minSize: 185,
  maxSize: 350,
  initialSize: 250,
  tooltip: "Clusters",
};
