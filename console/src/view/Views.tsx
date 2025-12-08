// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, type view } from "@synnaxlabs/client";
import {
  Component,
  Flex,
  type Flux,
  Form,
  Icon,
  List,
  Menu as PMenu,
  Select,
  Text,
  View,
} from "@synnaxlabs/pluto";
import { array, caseconv } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement, useCallback, useMemo } from "react";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { useConfirmDelete } from "@/ontology/hooks";
import { useContext } from "@/view/context";

export const Views = (): ReactElement | null => {
  const { resourceType, save, defaultViewKey } = useContext("View.Views");
  const remote = View.useList({ initialQuery: { types: [resourceType] } });
  const data = useMemo(
    () => [
      {
        key: defaultViewKey,
        name: `All ${caseconv.capitalize(plural(resourceType))}`,
        type: resourceType,
        query: {},
      },
    ],
    [resourceType, defaultViewKey],
  );
  const staticData = List.useStaticData<view.Key, view.View>({ data });
  const combinedData = List.useCombinedData<view.Key, view.View>({
    first: staticData,
    second: remote,
  });
  const { fetchMore } = List.usePager({
    retrieve: useCallback(
      (args) => {
        staticData.retrieve(args);
        remote.retrieve(args);
      },
      [staticData.retrieve, remote.retrieve],
    ),
  });
  const { set, reset } = Form.useContext();
  const selected = Form.useFieldValue<view.Key>("key", { optional: true });
  const handleSelectView = useCallback(
    (view: view.Key) => {
      const v = combinedData.getItem?.(view);
      if (v == null) return;
      set("", v);
      save();
    },
    [set, reset, combinedData.getItem],
  );
  const contextMenuProps = PMenu.useContextMenu();
  return (
    <PMenu.ContextMenu {...contextMenuProps} menu={contextMenu}>
      <Select.Frame<view.Key, view.View>
        {...combinedData}
        value={selected ?? undefined}
        onFetchMore={fetchMore}
        multiple={false}
        onChange={handleSelectView}
      >
        <List.Items<view.Key>
          className={CSS.BE("view", "views")}
          x
          align="center"
          gap="medium"
          onContextMenu={contextMenuProps.open}
        >
          {item}
        </List.Items>
      </Select.Frame>
    </PMenu.ContextMenu>
  );
};

interface ItemProps extends List.ItemProps<view.Key> {}

const Item = ({ itemKey }: ItemProps): ReactElement | null => {
  const item = List.useItem<view.Key, view.View>(itemKey);
  const { update: rename } = View.useRename();
  const handleRename = useCallback(
    (name: string) => rename({ key: itemKey, name }),
    [itemKey, rename],
  );
  if (item == null) return null;
  const { name } = item;
  return (
    <Flex.Box pack>
      <Select.Button itemKey={itemKey} size="small" justify="between">
        <Text.MaybeEditable
          id={itemKey}
          value={name}
          allowDoubleClick={false}
          color={7}
          onChange={handleRename}
          className={CSS.BE("view", "view-item")}
        />
      </Select.Button>
    </Flex.Box>
  );
};

const item = Component.renderProp(Item);

const ContextMenu = ({ keys }: PMenu.ContextMenuMenuProps): ReactElement | null => {
  const { defaultViewKey } = useContext("View.Views");
  const filteredKeys = keys.filter((k) => k !== defaultViewKey);
  const confirm = useConfirmDelete({
    icon: "Delete",
    type: "View",
    description: "Deleting this view will permanently remove it.",
  });
  const { getItem } = List.useUtilContext<view.Key, view.View>();
  const { get, reset } = Form.useContext();
  const { update: del } = View.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<View.DeleteParams>) => {
        const keys = array.toArray(data);
        if (keys.length === 0) throw new UnexpectedError("No views to delete");
        // we are only calling this with a single view so we can just use keys[0]
        const key = keys[0];
        const v = getItem?.(key);
        if (v == null) throw new UnexpectedError(`View with key ${key} not found`);
        const confirmed = await confirm([v]);
        if (!confirmed) return false;
        if (key === get<string>("key").value) reset();
        return key;
      },
      [getItem, confirm, get, reset],
    ),
  });
  const canRename = filteredKeys.length === 1;
  const canDelete = filteredKeys.length > 0;
  return (
    <PMenu.Menu level="small" gap="small">
      {canRename && (
        <PMenu.Item itemKey="rename" onClick={() => Text.edit(filteredKeys[0])}>
          <Icon.Rename />
          Rename
        </PMenu.Item>
      )}
      {canDelete && (
        <PMenu.Item itemKey="delete" onClick={() => del(filteredKeys[0])}>
          <Icon.Delete />
          Delete
        </PMenu.Item>
      )}
      {(canRename || canDelete) && <PMenu.Divider />}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};

const contextMenu = Component.renderProp(ContextMenu);
