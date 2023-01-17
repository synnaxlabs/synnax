// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space, Header, List, Text } from "@synnaxlabs/pluto";
import type { ListItemProps, NavDrawerItem } from "@synnaxlabs/pluto";
import clsx from "clsx";
import { AiOutlinePlus } from "react-icons/ai";
import { useDispatch } from "react-redux";

import { useSelectCluster, useSelectClusters } from "../store";
import { setActiveCluster } from "../store/slice";
import { RenderableCluster } from "../types";

import { ClusterIcon } from "./ClusterIcon";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout, useLayoutPlacer } from "@/features/layout";

import "./ClusterToolbar.css";

const connectClusterWindowLayout: Layout = {
  key: "connectCluster",
  type: "connectCluster",
  title: "Connect a Cluster",
  location: "window",
  window: {
    resizable: false,
    height: 430,
    width: 650,
    navTop: true,
  },
};

const Content = (): JSX.Element => {
  const dispatch = useDispatch();
  const data = Object.values(useSelectClusters());
  const active = useSelectCluster();
  const openWindow = useLayoutPlacer();

  const selected = active != null ? [active?.key] : [];

  const actions = [
    {
      children: <AiOutlinePlus />,
      onClick: () => openWindow(connectClusterWindowLayout),
    },
  ];

  const handleSelect = ([key]: readonly string[]): void => {
    dispatch(setActiveCluster(key ?? null));
  };

  return (
    <Space empty>
      <ToolbarHeader>
        <ToolbarTitle icon={<ClusterIcon />}>Clusters</ToolbarTitle>
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
    className={clsx(
      "delta-cluster-toolbar-list__item",
      selected && "delta-cluster-toolbar-list__item--selected"
    )}
    {...props}
  >
    <Text level="p">{name}</Text>
  </Space>
);

export const ClusterToolbar: NavDrawerItem = {
  key: "clusters",
  content: <Content />,
  icon: <ClusterIcon />,
  minSize: 185,
  maxSize: 350,
  initialSize: 250,
};
