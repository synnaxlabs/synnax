// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/remote/Toolbar.css";

import { type connection } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Divider,
  Dropdown as Core,
  List as CoreList,
  Menu as PMenu,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import {
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
} from "react";
import { useDispatch } from "react-redux";

import { Boundary } from "@/cluster/external";
import { connectWindowLayout } from "@/cluster/remote/Connect";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { type Cluster, remove, rename, setActive } from "@/cluster/slice";
import { Menu } from "@/components/menu";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";

export const List = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const dispatch = useDispatch();
  const allClusters = useSelectMany();
  const active = useSelect();
  const openWindow = Layout.usePlacer();
  const selected = active?.key ?? null;

  const handleConnect = (key: string | null): void => {
    dispatch(setActive(key));
  };

  const handleRemove = (key: string): void => {
    dispatch(remove({ keys: [key] }));
    if (key === active?.key) dispatch(setActive(null));
  };

  const handleRename = (key: string): void => Text.edit(`cluster-dropdown-${key}`);

  const handleLink = Link.useCopyToClipboard();

  const contextMenu = useCallback(
    ({ keys: [key] }: PMenu.ContextMenuMenuProps): ReactElement | null => {
      if (key == null) return <Layout.DefaultContextMenu />;

      const handleSelect = (menuKey: string): void => {
        switch (menuKey) {
          case "remove":
            return handleRemove(key);
          case "connect":
            return handleConnect(key);
          case "disconnect":
            return handleConnect(null);
          case "link": {
            const name = allClusters.find((c) => c.key === key)?.name;
            if (name == null) return;
            return handleLink({ clusterKey: key, name });
          }
          case "rename":
            return handleRename(key);
        }
      };

      return (
        <PMenu.Menu level="small" onChange={handleSelect}>
          {key === active?.key ? (
            <PMenu.Item
              startIcon={<Icon.Disconnect />}
              size="small"
              itemKey="disconnect"
            >
              Disconnect
            </PMenu.Item>
          ) : (
            <PMenu.Item startIcon={<Icon.Connect />} size="small" itemKey="connect">
              Connect
            </PMenu.Item>
          )}
          <Menu.RenameItem />
          <PMenu.Divider />
          <PMenu.Item startIcon={<Icon.Delete />} size="small" itemKey="remove">
            Remove
          </PMenu.Item>
          <Link.CopyMenuItem />
          <PMenu.Divider />
          <Menu.HardReloadItem />
        </PMenu.Menu>
      );
    },
    [active?.key, handleConnect, handleRemove],
  );

  return (
    <Align.Pack borderShade={4} className={CSS.B("cluster-list")} direction="y">
      <Align.Pack
        borderShade={4}
        direction="x"
        justify="spaceBetween"
        size="large"
        grow
      >
        <Align.Space
          className={CSS.B("cluster-list-title")}
          direction="y"
          justify="center"
          grow
        >
          <Text.WithIcon level="h5" startIcon={<Icon.Cluster />}>
            Clusters
          </Text.WithIcon>
        </Align.Space>
        <Button.Button
          variant="outlined"
          size="medium"
          startIcon={<Icon.Add />}
          onClick={() => openWindow(connectWindowLayout)}
          className={CSS.B("cluster-list-add")}
        >
          Add
        </Button.Button>
      </Align.Pack>
      <PMenu.ContextMenu
        style={{ width: "100%", height: 300 }}
        menu={contextMenu}
        {...menuProps}
      >
        <CoreList.List<string, Cluster> data={allClusters} emptyContent={<Boundary />}>
          <CoreList.Selector
            value={selected}
            allowMultiple={false}
            onChange={handleConnect}
          >
            <CoreList.Core<string, Cluster> style={{ height: "100%", width: "100%" }}>
              {({ key, ...p }) => <ListItem key={key} {...p} />}
            </CoreList.Core>
          </CoreList.Selector>
        </CoreList.List>
      </PMenu.ContextMenu>
    </Align.Pack>
  );
};

const ListItem = (props: CoreList.ItemProps<string, Cluster>): ReactElement => {
  const dispatch = useDispatch();
  const handleChange = (value: string) => {
    dispatch(rename({ key: props.entry.key, name: value }));
  };

  return (
    <CoreList.ItemFrame
      className={CSS(CSS.B("cluster-list-item"))}
      direction="x"
      align="center"
      {...props}
    >
      <Align.Space direction="y" justify="spaceBetween" size={0.5} grow>
        <Text.MaybeEditable
          level="p"
          id={`cluster-dropdown-${props.entry.key}`}
          weight={450}
          value={props.entry.name}
          onChange={handleChange}
          allowDoubleClick={false}
        />
        <Text.Text level="p" shade={6}>
          {props.entry.props.host}:{props.entry.props.port}
        </Text.Text>
      </Align.Space>
    </CoreList.ItemFrame>
  );
};

export const Dropdown = (): ReactElement | null => {
  const dropProps = Core.use();
  const cluster = useSelect();
  return (
    <Core.Dialog
      {...dropProps}
      variant="floating"
      bordered={false}
      className={CSS.B("cluster-dropdown")}
    >
      <Button.Button
        onClick={dropProps.toggle}
        variant="text"
        startIcon={<Icon.Cluster />}
        justify="center"
      >
        {cluster?.name ?? "No Active Cluster"}
      </Button.Button>
      <List />
    </Core.Dialog>
  );
};

/** Props for the ConnectionStateBadge component. */
export interface ConnectionStateBadgeProps {
  state: connection.State;
}

export const statusVariants: Record<connection.Status, Status.Variant> = {
  connected: "success",
  failed: "error",
  connecting: "loading",
  disconnected: "warning",
};

/**
 * A simple badge that displays the connection state of a cluster using an informative
 * text, icon, and color.
 * @param props - The props of the component.
 * @param props.state - The connection state of the cluster.
 */
export const ConnectionStatusBadge = ({
  state: { status },
}: ConnectionStateBadgeProps): ReactElement => (
  <Status.Text
    className={CSS.B("connection-status-badge")}
    variant={statusVariants[status]}
    justify="center"
  >
    {caseconv.capitalize(status)}
  </Status.Text>
);

/**
 * Displays the connection state of the cluster.
 */
export const ConnectionBadge = (): ReactElement => {
  const state = Synnax.useConnectionState();
  return <ConnectionStatusBadge state={state} />;
};

export const Toolbar = () => (
  <>
    <Divider.Divider />
    <Dropdown />
    <Divider.Divider />
    <ConnectionBadge />
  </>
);
