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
  Divider,
  Flex,
  Form,
  Icon,
  List,
  Menu,
  Select,
  Status,
  Text,
  View,
} from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Menu as CMenu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { useContext } from "@/view/context";

export const Views = (): ReactElement | null => {
  const { editable, resourceType } = useContext("View.Views");
  const listReturn = View.useList({ initialQuery: { types: [resourceType] } });
  const { retrieve, getItem } = listReturn;
  const { fetchMore } = List.usePager({ retrieve });
  const { set, reset, get } = Form.useContext();
  const selected = Form.useFieldValue<view.Key>("key", {
    optional: true,
  });
  const confirm = useConfirmDelete({
    type: "View",
    description: "Deleting this view will permanently remove it.",
  });
  const handleError = Status.useErrorHandler();
  const renameModal = Modals.useRename();
  const handleCreate = useCallback(() => {
    handleError(async () => {
      const name = await renameModal(
        { initialValue: `View for ${resourceType}` },
        { icon: "View", name: "View.Create" },
      );
      if (name == null) return;
      set("name", name);
      set("key", uuid.create());
    }, "Failed to create view");
  }, [handleError, resourceType]);
  const handleSelectView = useCallback(
    (view: view.Key | null) => {
      if (view == null) {
        reset();
        return;
      }
      const v = getItem(view);
      if (v == null) return;
      set("", v);
    },
    [set, reset, getItem],
  );
  const { update: del } = View.useDelete();
  const handleDelete = useCallback(
    (viewKey: view.Key) => {
      handleError(async () => {
        const v = getItem(viewKey);
        if (v == null) return;
        const confirmed = await confirm([v]);
        if (!confirmed) return;
        if (viewKey === get<string>("key").value) reset();
        del(viewKey);
      }, "Failed to delete view");
    },
    [reset, get, getItem, confirm, del, handleError],
  );
  const contextMenuProps = Menu.useContextMenu();
  return (
    <Select.Frame<view.Key, view.View>
      {...listReturn}
      value={selected ?? undefined}
      onFetchMore={fetchMore}
      multiple={false}
      onChange={handleSelectView}
      allowNone
    >
      <Flex.Box x style={{ padding: "1rem 1rem" }} gap="medium">
        <Text.Text align="center">
          Views
          {editable && (
            <Button.Button onClick={handleCreate} size="small" variant="shadow">
              <Icon.Add />
            </Button.Button>
          )}
        </Text.Text>
        <Divider.Divider y />
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
          <List.Items<view.Key> x gap="medium">
            {({ key, ...rest }) => (
              <ViewItem key={key} {...rest} onContextMenu={contextMenuProps.open} />
            )}
          </List.Items>
        </Menu.ContextMenu>
      </Flex.Box>
    </Select.Frame>
  );
};

interface ViewItemProps extends List.ItemProps<view.Key> {}

const ViewItem = ({ itemKey, onContextMenu }: ViewItemProps): ReactElement | null => {
  const query = View.useRetrieve({ key: itemKey });
  const { update: rename } = View.useRename();
  const handleRename = useCallback(
    (name: string) => {
      rename({ key: itemKey, name });
    },
    [itemKey, rename],
  );
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
    </Flex.Box>
  );
};
