// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/core/list/List.css";

import { connection } from "@synnaxlabs/client";
import {
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

import { Button } from "@/platform/button";
import { ContextMenu } from "@/platform/context-menu";
import { Item, nameID } from "@/platform/core/list/Item";
import { useConnectModal } from "@/platform/core/useConnectModal";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Link } from "@/platform/link";
import { Session } from "@/session";

export interface ListProps
  extends Input.Control<string | undefined>, Omit<Flex.BoxProps, "onChange"> {}

export const List = ({ value, onChange, ...rest }: ListProps): ReactElement => {
  const menuProps = Menu.useContextMenu();
  const dispatch = Session.useDispatch();
  const allCores = Session.Core.useSelectMany().sort((a, b) =>
    a.name.localeCompare(b.name),
  );
  const keys = useMemo(() => allCores.map((c) => c.key), [allCores]);
  const [testing, setTesting] = useState<string | null>(null);
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();

  const validateName = useCallback(
    (key: string, name: string): boolean => {
      if (!allCores.some((c) => c.name === name && c.key !== key)) return true;
      addStatus({
        variant: "error",
        message: `Failed to rename Core to ${name}`,
        description: `A Core with name "${name}" already exists.`,
      });
      return false;
    },
    [allCores, addStatus],
  );

  const handleRemove = (key: string): void => {
    if (key === value) {
      const nextCore = allCores.find((c) => c.key !== key);
      onChange(nextCore?.key);
    }
    dispatch(Session.Core.remove(key));
  };

  const handleRename = (key: string): void => Text.edit(nameID(key));

  const handleLink = Link.useCopyToClipboard();

  const openConnect = useConnectModal();

  const handleRetest = (key: string): void => {
    const core = allCores.find((c) => c.key === key);
    if (core == null) return;
    handleError(async () => {
      try {
        setTesting(key);
        const status = await connection.check({
          host: core.host,
          port: core.port,
          secure: core.secure,
          name: core.name,
        });
        if (status.variant === "success")
          addStatus({
            variant: "success",
            message: `Connected to ${core.name}`,
          });
        else
          addStatus({
            variant: "error",
            message: `Failed to connect to ${core.name}`,
            description: status.message,
          });
      } finally {
        setTesting(null);
      }
    }, `Failed to connect to ${core.name}`);
  };

  const handleEdit = (key: string): void => {
    openConnect({ coreKey: key });
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
          <Link.CopyContextMenuItem
            onClick={() => {
              const core = allCores.find((c) => c.key === key);
              if (core == null) return;
              handleLink({ clusterKey: core.clusterKey, name: core.name });
            }}
          />
          <Menu.Divider />
          <Menu.Item itemKey="remove" onClick={() => handleRemove(key)}>
            <Icon.Delete />
            Remove
          </Menu.Item>
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
        <Header.Header gap="small" x>
          <Header.Title level="h4" color={11}>
            <Icon.Core />
            Cores
          </Header.Title>
        </Header.Header>
        <Flex.Box
          empty
          onContextMenu={menuProps.open}
          grow
          className={CSS.cls(CSS.BE("shell", "items"), CSS.BE("core-list", "items"))}
        >
          {keys.length === 0 ? (
            <Empty.Action
              message="No Cores"
              action="Add Core"
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
        <Button.CreateListItem size="large" onClick={() => openConnect()}>
          Add a Core
        </Button.CreateListItem>
      </Flex.Box>
    </Select.Frame>
  );
};
