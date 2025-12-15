// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Flex,
  Header,
  Icon,
  type Input,
  Menu as PMenu,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { CONNECT_LAYOUT } from "@/cluster/Connect";
import { Item } from "@/cluster/list/Item";
import { useSelectMany } from "@/cluster/selectors";
import { remove } from "@/cluster/slice";
import { Menu } from "@/components";
import { Layout } from "@/layout";
import { Link } from "@/link";

export interface ListProps
  extends Input.Control<string | undefined>,
    Omit<Flex.BoxProps, "onChange"> {}

export const List = ({ value, onChange, ...rest }: ListProps): ReactElement => {
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
    if (key === value) {
      const nextCluster = allClusters.find((c) => c.key !== key);
      onChange(nextCluster?.key);
    }
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
      <Flex.Box y bordered grow empty {...rest}>
        <PMenu.ContextMenu menu={contextMenu} {...menuProps} />
        <Header.Header gap="small" x style={{ padding: "0.666rem" }}>
          <Header.Title level="h4" color={11}>
            <Icon.Cluster />
            Cores
          </Header.Title>
          <Button.Button onClick={() => placeLayout(CONNECT_LAYOUT)} variant="filled">
            <Icon.Add />
          </Button.Button>
        </Header.Header>
        <Flex.Box empty onContextMenu={menuProps.open} grow>
          {keys.map((key, i) => (
            <Item key={key} index={i} itemKey={key} validateName={validateName} />
          ))}
        </Flex.Box>
      </Flex.Box>
    </Select.Frame>
  );
};
