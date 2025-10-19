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
  Button,
  Dialog,
  Flex,
  Header,
  Icon,
  List as CoreList,
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
  type ReactNode,
  useCallback,
  useMemo,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { ConnectionBadge } from "@/cluster/Badges";
import { CONNECT_LAYOUT } from "@/cluster/Connect";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { remove, rename, setActive } from "@/cluster/slice";
import { EmptyAction, Menu } from "@/components";
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
  const { selected, onSelect } = Select.useItemState(rest.itemKey);
  const handleChange = (value: string) => {
    if (!validateName(value) || item == null) return;
    dispatch(rename({ key: item.key, name: value }));
  };

  if (item == null) return null;
  return (
    <CoreList.Item
      className={CSS(CSS.B("cluster-list-item"))}
      y
      selected={selected}
      onSelect={onSelect}
      gap="small"
      {...rest}
    >
      <Text.MaybeEditable
        id={`cluster-dropdown-${item.key}`}
        weight={500}
        value={item.name}
        onChange={handleChange}
        allowDoubleClick={false}
      />
      <Text.Text color={9} weight={450}>
        {item.host}:{item.port}
      </Text.Text>
    </CoreList.Item>
  );
};

export interface NoneConnectedBoundaryProps extends PropsWithChildren {}

export const NoneConnectedBoundary = ({
  children,
}: NoneConnectedBoundaryProps): ReactNode => {
  const client = Synnax.use();
  if (client != null) return children;
  return <NoneConnected />;
};

export interface NoneConnectedProps extends Flex.BoxProps<"div"> {}

export const NoneConnected = (props: NoneConnectedProps): ReactElement => {
  const placeLayout = Layout.usePlacer();

  const handleCluster: Text.TextProps["onClick"] = (e: MouseEvent) => {
    e.stopPropagation();
    placeLayout(CONNECT_LAYOUT);
  };

  return (
    <EmptyAction
      message="No Core connected."
      action="Connect a Core"
      onClick={handleCluster}
      {...props}
    />
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
        message: `Cannot rename Core to ${name}`,
        description: `A Core with name "${name}" already exists.`,
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
            <PMenu.Item size="small" itemKey="disconnect">
              <Icon.Disconnect />
              Disconnect
            </PMenu.Item>
          ) : (
            <PMenu.Item size="small" itemKey="connect">
              <Icon.Connect />
              Connect
            </PMenu.Item>
          )}
          <Menu.RenameItem />
          <PMenu.Divider />
          <PMenu.Item size="small" itemKey="remove">
            <Icon.Delete />
            Remove
          </PMenu.Item>
          <Link.CopyMenuItem />
          <PMenu.Divider />
          <Menu.ReloadConsoleItem />
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
        <Flex.Box pack>
          <Dialog.Trigger
            justify="center"
            contrast={2}
            variant={disconnected ? "filled" : "outlined"}
            hideCaret
          >
            {disconnected ? <Icon.Connect /> : <Icon.Cluster />}
            {cluster?.name ?? "Connect a Core"}
          </Dialog.Trigger>
          <ConnectionBadge />
        </Flex.Box>
        <Dialog.Dialog style={{ minWidth: 300, width: 400 }} bordered borderColor={6}>
          <PMenu.ContextMenu menu={contextMenu} {...menuProps} />
          <Flex.Box pack x>
            <Header.Header grow borderColor={6} gap="small" x>
              <Header.Title level="h5">
                <Icon.Cluster />
                Cores
              </Header.Title>
            </Header.Header>
            <Button.Button
              variant="filled"
              size="large"
              gap="small"
              onClick={() => {
                placeLayout(CONNECT_LAYOUT);
                setDialogVisible(false);
              }}
              className={CSS.B("cluster-list-add")}
            >
              <Icon.Connect />
              Connect
            </Button.Button>
          </Flex.Box>
          <Flex.Box
            className={CSS.B("cluster-list")}
            empty
            style={{ height: 190 }}
            onContextMenu={menuProps.open}
          >
            {keys.map((key, i) => (
              <ListItem key={key} index={i} itemKey={key} validateName={validateName} />
            ))}
          </Flex.Box>
        </Dialog.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
