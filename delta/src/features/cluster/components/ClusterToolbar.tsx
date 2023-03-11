// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Space, Header, List, Text } from "@synnaxlabs/pluto";
import type { ListItemProps } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { connectClusterWindowLayout } from "@/features/cluster/components/ConnectCluster";
import {
  setActiveCluster,
  useSelectCluster,
  useSelectClusters,
} from "@/features/cluster/store";
import { RenderableCluster } from "@/features/cluster/types";
import { useLayoutPlacer, NavDrawerItem } from "@/features/layout";

import "./ClusterToolbar.css";

const Content = (): JSX.Element => {
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

  const handleSelect = ([key]: readonly string[]): void => {
    dispatch(setActiveCluster(key ?? null));
  };

  return (
    <Space empty>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Cluster />}>Clusters</ToolbarTitle>
        <Header.Actions>{actions}</Header.Actions>
      </ToolbarHeader>
      <List<RenderableCluster> data={data}>
        <List.Selector value={selected} onChange={handleSelect} allowMultiple={false} />
        <List.Core.Virtual itemHeight={30}>{ListItem}</List.Core.Virtual>
      </List>
    </Space>
  );
};

const ListItem = ({
  entry: { key, name },
  style,
  selected,
  onSelect,
  ...props
}: ListItemProps<RenderableCluster>): JSX.Element => (
  <Space
    direction="x"
    align="center"
    justify="spaceBetween"
    onDoubleClick={() => onSelect?.(key)}
    className={CSS(
      CSS.BE("cluster-toolbar-list", "item"),
      selected && CSS.M("selected")
    )}
    {...props}
  >
    <Text level="p">{name}</Text>
  </Space>
);

/** Configuration and content for the cluster nav drawer toolbar. */
export const ClusterToolbar: NavDrawerItem = {
  key: "clusters",
  content: <Content />,
  icon: <Icon.Cluster />,
  minSize: 185,
  maxSize: 350,
  initialSize: 250,
};
