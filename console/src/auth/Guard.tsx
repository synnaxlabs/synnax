// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Dropdown.css";

import {
  Button,
  Flex,
  Header,
  Icon,
  type Input,
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
} from "react";
import { useDispatch } from "react-redux";

import { Login } from "@/auth/Login";
import { CONNECT_LAYOUT } from "@/cluster/Connect";
import { useSelect, useSelectMany } from "@/cluster/selectors";
import { remove, rename } from "@/cluster/slice";
import { EmptyAction, Menu } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";

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

export interface ClusterListProps extends Input.Control<string | undefined> {}

export const ClusterList = ({ value, onChange }: ClusterListProps): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const dispatch = useDispatch();
  const allClusters = useSelectMany().sort((a, b) => a.name.localeCompare(b.name));
  const keys = useMemo(() => allClusters.map((c) => c.key), [allClusters]);
  const addStatus = Status.useAdder();

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
    [handleRemove],
  );

  const placeLayout = Layout.usePlacer();

  return (
    <Select.Frame data={keys} value={value} onChange={onChange} itemHeight={54}>
      <Flex.Box y bordered grow empty>
        <PMenu.ContextMenu menu={contextMenu} {...menuProps} />
        <Header.Header gap="small" x style={{ padding: "0.666rem" }}>
          <Header.Title level="h4" color={11}>
            <Icon.Cluster />
            Cores
          </Header.Title>
          <Button.Button onClick={() => placeLayout(CONNECT_LAYOUT)} variant="filled">
            <Icon.Connect />
          </Button.Button>
        </Header.Header>
        <Flex.Box empty onContextMenu={menuProps.open} grow>
          {keys.map((key, i) => (
            <ListItem key={key} index={i} itemKey={key} validateName={validateName} />
          ))}
        </Flex.Box>
      </Flex.Box>
    </Select.Frame>
  );
};

export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const active = useSelect();
  if (active != null) return children;
  return <Login />;
};
