// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align, Header, List, Text, componentRenderProp } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { connectWindowLayout } from "@/cluster/Connect";
import { type RenderableCluster } from "@/cluster/core";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { setActive } from "@/cluster/slice";
import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";

import "@/cluster/Toolbar.css";

const Content = (): ReactElement => {
  const dispatch = useDispatch();
  const data = Object.values(useSelectMany());
  const active = useSelect();
  const openWindow = Layout.usePlacer();

  const selected = active != null ? [active?.key] : [];

  const actions = [
    {
      children: <Icon.Add />,
      onClick: () => openWindow(connectWindowLayout),
    },
  ];

  const handleSelect = ([key]: string[]): void => {
    dispatch(setActive(key ?? null));
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
export const Toolbar: Layout.NavDrawerItem = {
  key: "clusters",
  content: <Content />,
  icon: <Icon.Cluster />,
  minSize: 185,
  maxSize: 350,
  initialSize: 250,
  tooltip: "Clusters",
};
