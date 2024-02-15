// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Header,
  List,
  Synnax,
  Text,
  componentRenderProp,
  Menu as PMenu,
} from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { connectWindowLayout } from "@/cluster/Connect";
import { type RenderableCluster } from "@/cluster/core";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { remove, setActive } from "@/cluster/slice";
import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { setNavdrawerVisible } from "@/layout/slice";

import "@/cluster/Toolbar.css";

import { LOCAL_KEY } from "./local";

const Content = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
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

  const handleConnect = (key: string | null): void => {
    dispatch(setActive(key));
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };

  const ContextMenu = ({ keys: [key] }: PMenu.ContextMenuProps): ReactElement => {
    const handleSelect = (menuKey: string): void => {
      if (key == null) return;
      switch (menuKey) {
        case "remove":
          return handleRemove([key]);
        case "connect":
          return handleConnect([key]);
        case "disconnect":
          return handleConnect([]);
      }
    };

    return (
      <PMenu.Menu onChange={handleSelect}>
        {key !== LOCAL_KEY && (
          <PMenu.Item startIcon={<Icon.Delete />} size="small" itemKey="remove">
            Remove
          </PMenu.Item>
        )}
        {key === active?.key ? (
          <PMenu.Item startIcon={<Icon.Disconnect />} size="small" itemKey="disconnect">
            Disconnect
          </PMenu.Item>
        ) : (
          <PMenu.Item startIcon={<Icon.Connect />} size="small" itemKey="connect">
            Connect
          </PMenu.Item>
        )}
      </PMenu.Menu>
    );
  };

  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Cluster />}>Clusters</ToolbarTitle>
        <Header.Actions>{actions}</Header.Actions>
      </ToolbarHeader>
      <PMenu.ContextMenu menu={(props) => <ContextMenu {...props} />} {...menuProps}>
        <List.List<string, RenderableCluster>
          data={data}
          emptyContent={<NoneConnected />}
        >
          <List.Selector
            value={selected}
            onChange={handleConnect}
            allowMultiple={false}
          />
          <List.Core.Virtual itemHeight={30}>
            {componentRenderProp(ListItem)}
          </List.Core.Virtual>
        </List.List>
      </PMenu.ContextMenu>
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
    id={key}
    direction="x"
    align="center"
    justify="spaceBetween"
    onDoubleClick={() => onSelect?.(key)}
    className={CSS(
      CSS.BE("cluster-toolbar-list", "item"),
      selected && CSS.M("selected"),
      PMenu.CONTEXT_TARGET,
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

export interface NoneConnectedProps extends PropsWithChildren {}

export const NoneConnectedBoundary = ({
  children,
}: NoneConnectedProps): ReactElement => {
  const client = Synnax.use();
  if (client != null) return <>{children}</>;
  return <NoneConnected />;
};

export const NoneConnected = (): ReactElement => {
  const dispatch = useDispatch();
  const placer = Layout.usePlacer();
  const handleCluster: Text.TextProps["onClick"] = (e) => {
    e.stopPropagation();
    placer(connectWindowLayout);
    dispatch(setNavdrawerVisible({ key: Toolbar.key, value: true }));
  };
  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <Align.Center direction="y" style={{ height: "100%" }} size="small">
        <Text.Text level="p">No cluster connected.</Text.Text>
        <Text.Link level="p" onClick={handleCluster}>
          Connect a cluster
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};
