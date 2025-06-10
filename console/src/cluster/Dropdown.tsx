// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Dropdown.css";

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Dropdown as Core,
  Header,
  List as CoreList,
  Menu as PMenu,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import {
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
} from "react";
import { useDispatch } from "react-redux";

import { ConnectionBadge } from "@/cluster/Badges";
import { CONNECT_LAYOUT } from "@/cluster/Connect";
import { getClient } from "@/cluster/getClient";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { type Cluster, remove, rename, setActive } from "@/cluster/slice";
import { Menu } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { useCreateOrRetrieve } from "@/workspace/useCreateNew";

export const List = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const dispatch = useDispatch();
  const allClusters = useSelectMany().sort((a, b) => a.name.localeCompare(b.name));
  const active = useSelect();
  const placeLayout = Layout.usePlacer();
  const selected = active?.key ?? null;
  const addStatus = Status.useAdder();
  const createWS = useCreateOrRetrieve();

  const handleConnect = (key: string | null): void => {
    dispatch(setActive(key));
    const cluster = allClusters.find((c) => c.key === key);
    createWS(cluster == null ? null : getClient(cluster));
  };

  const validateName = useCallback(
    (name: string): boolean => {
      const allNames = allClusters.map((c) => c.name);
      if (!allNames.includes(name)) return true;
      addStatus({
        variant: "error",
        message: `Cannot rename cluster to ${name}`,
        description: `A cluster with name "${name}" already exists.`,
      });
      return false;
    },
    [allClusters, addStatus],
  );

  const handleRemove = (key: string): void => {
    dispatch(remove(key));
    if (key === active?.key) dispatch(setActive(null));
  };

  const handleRename = (key: string): void => Text.edit(`cluster-dropdown-${key}`);

  const handleLink = Link.useCopyToClipboard();

  const contextMenu = useCallback(
    ({ keys: [key] }: PMenu.ContextMenuMenuProps): ReactElement => {
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
    <Align.Pack className={CSS.B("cluster-list")} y>
      <Align.Pack x justify="spaceBetween" size="large" grow>
        <Header.Header grow bordered borderShade={5} size="small">
          <Header.Title level="h5" startIcon={<Icon.Cluster />}>
            Clusters
          </Header.Title>
        </Header.Header>
        <Button.Button
          variant="filled"
          size="large"
          iconSpacing="small"
          startIcon={<Icon.Connect />}
          onClick={() => placeLayout(CONNECT_LAYOUT)}
          className={CSS.B("cluster-list-add")}
        >
          Connect
        </Button.Button>
      </Align.Pack>
      <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
        <CoreList.List<string, Cluster>
          data={allClusters}
          emptyContent={<NoneConnected />}
        >
          <CoreList.Selector
            value={selected}
            allowMultiple={false}
            onChange={handleConnect}
          >
            <CoreList.Core<string, Cluster>
              style={{ height: 190, width: "100%" }}
              onContextMenu={menuProps.open}
              className={menuProps.className}
              bordered
              borderShade={5}
            >
              {({ key, ...p }) => (
                <ListItem key={key} {...p} validateName={validateName} />
              )}
            </CoreList.Core>
          </CoreList.Selector>
        </CoreList.List>
      </PMenu.ContextMenu>
    </Align.Pack>
  );
};

interface ListItemProps extends CoreList.ItemProps<string, Cluster> {
  validateName: (name: string) => boolean;
}

const ListItem = ({ validateName, ...rest }: ListItemProps): ReactElement => {
  const dispatch = useDispatch();
  const handleChange = (value: string) => {
    if (!validateName(value)) return;
    dispatch(rename({ key: rest.entry.key, name: value }));
  };

  return (
    <CoreList.ItemFrame
      className={CSS(CSS.B("cluster-list-item"))}
      x
      align="center"
      {...rest}
    >
      <Align.Space y justify="spaceBetween" size="tiny" grow>
        <Text.MaybeEditable
          level="p"
          id={`cluster-dropdown-${rest.entry.key}`}
          weight={450}
          value={rest.entry.name}
          onChange={handleChange}
          allowDoubleClick={false}
        />
        <Text.Text level="p" shade={10}>
          {rest.entry.host}:{rest.entry.port}
        </Text.Text>
      </Align.Space>
    </CoreList.ItemFrame>
  );
};

export interface NoneConnectedProps extends PropsWithChildren {}

export const NoneConnectedBoundary = ({
  children,
  ...rest
}: NoneConnectedProps): ReactElement => {
  const client = Synnax.use();
  if (client != null) return <>{children}</>;
  return <NoneConnected {...rest} />;
};

export interface NoneConnectedProps extends Align.SpaceProps<"div"> {}

export const NoneConnected = ({ style, ...rest }: NoneConnectedProps): ReactElement => {
  const placeLayout = Layout.usePlacer();

  const handleCluster: Text.TextProps["onClick"] = (e: MouseEvent) => {
    e.stopPropagation();
    placeLayout(CONNECT_LAYOUT);
  };

  return (
    <Align.Space
      empty
      style={{ height: "100%", position: "relative", ...style }}
      {...rest}
    >
      <Align.Center y style={{ height: "100%" }} size="small">
        <Text.Text level="p">No cluster connected.</Text.Text>
        <Text.Link level="p" onClick={handleCluster}>
          Connect a cluster
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};

export const Dropdown = (): ReactElement => {
  const { close, toggle, visible } = Core.use();
  const cluster = useSelect();
  const disconnected = cluster == null;
  return (
    <Align.Pack>
      <Core.Dialog
        close={close}
        visible={visible}
        variant="floating"
        bordered={false}
        className={CSS.B("cluster-dropdown")}
        borderShade={5}
        rounded={0.5}
      >
        <Button.Button
          onClick={toggle}
          startIcon={disconnected ? <Icon.Connect /> : <Icon.Cluster />}
          justify="center"
          shade={2}
          variant={disconnected ? "filled" : "outlined"}
        >
          {cluster?.name ?? "Connect Cluster"}
        </Button.Button>
        <List />
      </Core.Dialog>
      <ConnectionBadge />
    </Align.Pack>
  );
};
