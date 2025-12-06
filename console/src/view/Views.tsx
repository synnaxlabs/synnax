// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Form,
  Icon,
  List,
  Menu,
  Select,
  Text,
  View,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { Menu as CMenu } from "@/components";
import { useContext } from "@/view/context";

export const Views = (): ReactElement | null => {
  const { editable, resourceType } = useContext("View.Views");
  const listReturn = View.useList({ initialQuery: { types: [resourceType] } });
  const { fetchMore } = List.usePager({ retrieve: listReturn.retrieve });
  const { set, reset } = Form.useContext();
  const selected = Form.useFieldValue<view.Key>("key", {
    optional: true,
  });
  const handleSelectView = useCallback(
    (view: view.Key | null) => {
      if (view == null) {
        reset();
        return;
      }
      const v = listReturn.getItem(view);
      if (v == null) return;
      set("", v);
    },
    [set, reset, listReturn.getItem],
  );
  const { update: del } = View.useDelete();
  const handleDelete = useCallback(
    (viewKey: view.Key) => {
      if (viewKey === selected) reset();

      del(viewKey);
    },
    [reset, selected],
  );
  const contextMenuProps = Menu.useContextMenu();
  if (!editable) return null;
  return (
    <Select.Frame<view.Key, view.View>
      {...listReturn}
      value={selected ?? undefined}
      onFetchMore={fetchMore}
      multiple={false}
      onChange={handleSelectView}
      allowNone
    >
      <Menu.ContextMenu
        {...contextMenuProps}
        menu={({ keys }) => (
          <Menu.Menu level="small" gap="small">
            <Menu.Item
              itemKey="rename"
              onClick={() => {
                Text.edit(keys[0]);
              }}
            >
              <Icon.Rename />
              Rename
            </Menu.Item>
            <Menu.Item itemKey="delete" onClick={() => handleDelete(keys[0])}>
              <Icon.Delete />
              Delete
            </Menu.Item>
            <Menu.Divider />
            <CMenu.ReloadConsoleItem />
          </Menu.Menu>
        )}
      >
        <List.Items<view.Key> x gap="medium" style={itemsStyle}>
          {({ key, ...rest }) => (
            <ViewItem
              key={key}
              onDelete={handleDelete}
              {...rest}
              onContextMenu={contextMenuProps.open}
            />
          )}
        </List.Items>
      </Menu.ContextMenu>
    </Select.Frame>
  );
};

const itemsStyle = { padding: "1rem 1.5rem", overflow: "scroll" } as const;

interface ViewItemProps extends List.ItemProps<view.Key> {
  onDelete: (viewKey: view.Key) => void;
}

const ViewItem = ({
  itemKey,
  onDelete,
  onContextMenu,
}: ViewItemProps): ReactElement | null => {
  const query = View.useRetrieve({ key: itemKey });
  const { update: rename } = View.useRename();
  const handleRename = useCallback(
    (name: string) => {
      rename({ key: itemKey, name });
    },
    [itemKey, rename],
  );
  const handleDelete = useCallback(() => {
    onDelete(itemKey);
  }, [itemKey, onDelete]);
  if (query.variant !== "success") return null;
  return (
    <Flex.Box pack onContextMenu={onContextMenu}>
      <Select.Button itemKey={itemKey} size="small" justify="between">
        <Text.MaybeEditable
          id={`text-${itemKey}`}
          value={query.data.name}
          onChange={handleRename}
          style={{ padding: "0rem 1rem" }}
        />
      </Select.Button>
      <Button.Button onClick={handleDelete} size="small">
        <Icon.Delete />
      </Button.Button>
    </Flex.Box>
  );
};
