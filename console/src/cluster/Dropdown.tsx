// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Dropdown.css";

import { Synnax as Client } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Dialog,
  Header,
  Icon,
  type List as CoreList,
  Menu as PMenu,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import {
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { ConnectionBadge } from "@/cluster/Badges";
import { CONNECT_LAYOUT } from "@/cluster/Connect";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { remove, rename, setActive } from "@/cluster/slice";
import { Menu } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { clear } from "@/workspace/slice";
import { useCreateOrRetrieve } from "@/workspace/useCreateOrRetrieve";

interface ListItemProps extends CoreList.ItemProps<string> {
  validateName: (name: string) => boolean;
}

const ListItem = ({ validateName, ...rest }: ListItemProps): ReactElement | null => {
  const dispatch = useDispatch();
  const item = useSelect(rest.itemKey);
  const handleChange = (value: string) => {
    if (!validateName(value) || item == null) return;
    dispatch(rename({ key: item.key, name: value }));
  };

  if (item == null) return null;
  return (
    <Select.ListItem
      className={CSS(CSS.B("cluster-list-item"))}
      x
      align="center"
      {...rest}
    >
      <Align.Space y justify="spaceBetween" gap="tiny" grow>
        <Text.MaybeEditable
          level="p"
          id={`cluster-dropdown-${item.key}`}
          weight={450}
          value={item.name}
          onChange={handleChange}
          allowDoubleClick={false}
        />
        <Text.Text level="p" shade={10}>
          {item.host}:{item.port}
        </Text.Text>
      </Align.Space>
    </Select.ListItem>
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
      <Align.Center y style={{ height: "100%" }} gap="small">
        <Text.Text level="p">No cluster connected.</Text.Text>
        <Text.Link level="p" onClick={handleCluster}>
          Connect a cluster
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};

export const Dropdown = (): ReactElement => {
  const [dialogVisible, setDialogVisible] = useState(false);
  const cluster = useSelect();
  const disconnected = cluster == null;
  const menuProps = PMenu.useContextMenu();
  const dispatch = useDispatch();
  const allClusters = useSelectMany().sort((a, b) => a.name.localeCompare(b.name));
  const keys = useMemo(() => allClusters.map((c) => c.key), [allClusters]);
  const active = useSelect();
  const placeLayout = Layout.usePlacer();
  const selected = active?.key;
  const addStatus = Status.useAdder();
  const createWS = useCreateOrRetrieve();

  const handleConnect = (key: string | null): void => {
    dispatch(setActive(key));
    const cluster = allClusters.find((c) => c.key === key);
    if (cluster == null) {
      dispatch(clear());
      return;
    }
    createWS(new Client(cluster));
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
    <Dialog.Frame visible={dialogVisible} onVisibleChange={setDialogVisible}>
      <Select.Frame
        data={keys}
        value={selected}
        onChange={handleConnect}
        itemHeight={54}
        allowNone
      >
        <Align.Pack>
          <Dialog.Trigger
            startIcon={disconnected ? <Icon.Connect /> : <Icon.Cluster />}
            justify="center"
            shade={2}
            variant={disconnected ? "filled" : "outlined"}
            hideCaret
          >
            {cluster?.name ?? "Connect Cluster"}
          </Dialog.Trigger>
          <ConnectionBadge />
        </Align.Pack>
        <Dialog.Dialog style={{ minWidth: 300, width: 400 }}>
          <PMenu.ContextMenu menu={contextMenu} {...menuProps} />
          <Align.Pack x>
            <Header.Header grow bordered borderShade={6} gap="small" x>
              <Header.Title level="h5" startIcon={<Icon.Cluster />}>
                Clusters
              </Header.Title>
            </Header.Header>
            <Button.Button
              variant="filled"
              size="large"
              gap="small"
              startIcon={<Icon.Connect />}
              onClick={() => {
                placeLayout(CONNECT_LAYOUT);
                setDialogVisible(false);
              }}
              className={CSS.B("cluster-list-add")}
            >
              Connect
            </Button.Button>
          </Align.Pack>
          <Align.Space empty bordered borderShade={6} style={{ height: 190 }}>
            {keys.map((key, i) => (
              <ListItem key={key} index={i} itemKey={key} validateName={validateName} />
            ))}
          </Align.Space>
        </Dialog.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
