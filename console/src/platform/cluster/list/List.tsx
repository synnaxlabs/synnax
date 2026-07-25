// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/cluster/list/List.css";

import { checkConnection } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Header,
  Icon,
  type Input,
  Menu,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { Item } from "@/platform/cluster/list/Item";
import { useConnectModal } from "@/platform/cluster/useConnectModal";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Link } from "@/platform/link";
import { Session } from "@/session";

export interface ListProps
  extends Input.Control<string | undefined>, Omit<Flex.BoxProps, "onChange"> {}

export const List = ({ value, onChange, ...rest }: ListProps): ReactElement => {
  const menuProps = Menu.useContextMenu();
  const dispatch = Session.useDispatch();
  const allClusters = Session.Cluster.useSelectMany().sort((a, b) =>
    a.name.localeCompare(b.name),
  );
  const keys = useMemo(() => allClusters.map((c) => c.key), [allClusters]);
  const [testing, setTesting] = useState<string | null>(null);
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();

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
    dispatch(Session.Cluster.remove(key));
  };

  const handleRename = (key: string): void => Text.edit(`cluster-dropdown-${key}`);

  const handleLink = Link.useCopyToClipboard();

  const openConnect = useConnectModal();

  const handleRetest = (key: string): void => {
    const cluster = allClusters.find((c) => c.key === key);
    if (cluster == null) return;
    handleError(async () => {
      try {
        setTesting(key);
        const status = await checkConnection({
          host: cluster.host,
          port: cluster.port,
          secure: cluster.secure,
          name: cluster.name,
        });
        if (status.variant === "success") {
          addStatus({
            variant: "success",
            message: `Connected to ${cluster.name}`,
          });
          if (status.details.clusterKey && status.details.clusterKey !== key)
            dispatch(
              Session.Cluster.changeKey({
                oldKey: key,
                newKey: status.details.clusterKey,
              }),
            );
        } else
          addStatus({
            variant: "error",
            message: `Failed to connect to ${cluster.name}`,
            description: status.message,
          });
      } finally {
        setTesting(null);
      }
    }, `Failed to connect to ${cluster.name}`);
  };

  const handleEdit = (key: string): void => {
    openConnect({ clusterKey: key });
  };

  const contextMenu = useCallback(
    ({ keys: [key] }: Menu.ContextMenuMenuProps): ReactElement => {
      if (key == null)
        return (
          <ContextMenu.Menu>
            <ContextMenu.ReloadConsoleItem />
          </ContextMenu.Menu>
        );

      return (
        <ContextMenu.Menu>
          <ContextMenu.RenameItem onClick={() => handleRename(key)} />
          <Menu.Item itemKey="edit" onClick={() => handleEdit(key)}>
            <Icon.Edit />
            Edit
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item itemKey="retest" onClick={() => handleRetest(key)}>
            <Icon.Refresh />
            Refresh connection
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item itemKey="remove" onClick={() => handleRemove(key)}>
            <Icon.Delete />
            Remove
          </Menu.Item>
          <Link.CopyContextMenuItem
            onClick={() => {
              const name = allClusters.find((c) => c.key === key)?.name;
              if (name == null) return;
              handleLink({ clusterKey: key, name });
            }}
          />
          <Menu.Divider />
          <ContextMenu.ReloadConsoleItem />
        </ContextMenu.Menu>
      );
    },
    [handleRemove, handleRetest, handleEdit],
  );

  return (
    <Select.Frame data={keys} value={value} onChange={onChange} itemHeight={54}>
      <Flex.Box y bordered grow empty {...rest}>
        <Menu.ContextMenu menu={contextMenu} {...menuProps} />
        <Header.Header gap="small" x className={CSS.BE("cluster-list", "header")}>
          <Header.Title level="h4" color={11}>
            <Icon.Cluster />
            Cores
          </Header.Title>
          <Button.Button onClick={() => openConnect()} variant="filled">
            <Icon.Add />
          </Button.Button>
        </Header.Header>
        <Flex.Box empty onContextMenu={menuProps.open} grow>
          {keys.length === 0 ? (
            <Empty.Action
              message="No Cores added."
              action="Add a Core"
              onClick={() => openConnect()}
            />
          ) : (
            keys.map((key, i) => (
              <Item
                key={key}
                index={i}
                itemKey={key}
                validateName={validateName}
                loading={testing === key}
              />
            ))
          )}
        </Flex.Box>
      </Flex.Box>
    </Select.Frame>
  );
};
