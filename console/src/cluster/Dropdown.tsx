// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Dropdown.css";

import { Synnax as CSynnax, user } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Dropdown as Core,
  List as CoreList,
  Menu as PMenu,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import {
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
} from "react";
import { useDispatch } from "react-redux";

import { connectWindowLayout } from "@/cluster/Connect";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { type Cluster, remove, rename, setActive } from "@/cluster/slice";
import { Menu } from "@/components/menu";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Permissions } from "@/permissions";

const useHandleConnect = (): ((key: string | null) => void) => {
  const dispatch = useDispatch();
  const addStatus = Status.useAggregator();
  const allClusters = useSelectMany();
  return useMutation<void, Error, string | null>({
    onMutate: (key) => {
      dispatch(setActive(key));
      dispatch(Permissions.giveAllPermissions());
    },
    mutationFn: async (key) => {
      if (key == null) return;
      const cluster = allClusters.find((c) => c.key === key);
      if (cluster == null) throw new Error(`Cluster with key ${key} not found`);
      const client = new CSynnax(cluster.props);
      const username = client.props.username;
      const user_ = await client.user.retrieveByName(username);
      const policies = await client.access.policy.retrieveFor(
        user.ontologyID(user_.key),
      );
      dispatch(Permissions.set({ policies }));
    },
    onError: (error) => {
      dispatch(Permissions.clear());
      addStatus({
        variant: "error",
        message: "Failed to connect to cluster",
        description: error.message,
      });
    },
  }).mutate;
};

export const List = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const dispatch = useDispatch();
  const allClusters = useSelectMany();
  const active = useSelect();
  const openWindow = Layout.usePlacer();
  const selected = active?.key ?? null;
  const handleConnect = useHandleConnect();

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
    if (active != null && keys.includes(active?.key)) {
      dispatch(setActive(null));
      dispatch(Permissions.giveAllPermissions());
    }
  };

  const handleRename = (key: string): void => Text.edit(`cluster-dropdown-${key}`);

  const handleLink = Link.useCopyToClipboard();

  const contextMenu = useCallback(
    ({ keys: [key] }: PMenu.ContextMenuMenuProps): ReactElement | null => {
      if (key == null) return <Layout.DefaultContextMenu />;

      const handleSelect = (menuKey: string): void => {
        switch (menuKey) {
          case "remove":
            return handleRemove([key]);
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
    <Align.Pack className={CSS.B("cluster-list")} direction="y">
      <Align.Pack direction="x" justify="spaceBetween" size="large" grow>
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
        <CoreList.List<string, Cluster>
          data={allClusters}
          emptyContent={<NoneConnected />}
        >
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
  const handleChange = (value: string) =>
    void dispatch(rename({ key: props.entry.key, name: value }));

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

export interface NoneConnectedProps extends PropsWithChildren {}

export const NoneConnectedBoundary = ({
  children,
}: NoneConnectedProps): ReactElement => {
  const client = Synnax.use();
  if (client != null) return <>{children}</>;
  return <NoneConnected />;
};

export const NoneConnected = (): ReactElement => {
  const placer = Layout.usePlacer();

  const handleCluster: Text.TextProps["onClick"] = (e: MouseEvent) => {
    e.stopPropagation();
    placer(connectWindowLayout);
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

export const Dropdown = (): ReactElement => {
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
