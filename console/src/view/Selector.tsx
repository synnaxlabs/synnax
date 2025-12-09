// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, view } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  Flex,
  type Flux,
  Icon,
  List,
  Menu as PMenu,
  Select,
  Text,
  View as PView,
} from "@synnaxlabs/pluto";
import { location, uuid } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement, useCallback } from "react";

import { Controls, Menu } from "@/components";
import { CSS } from "@/css";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";
import { useContext, type View } from "@/view/context";

export const Selector = (): ReactElement | null => {
  const {
    editable,
    setEditable,
    resourceType,
    staticViews,
    selected,
    select: onSelect,
  } = useContext("View.Selector");
  const staticProps = List.useStaticData<view.Key, View>({ data: staticViews });
  const remoteProps = PView.useList({ initialQuery: { types: [resourceType] } });
  const { retrieve } = remoteProps;
  const combinedProps = List.useCombinedData<view.Key, View>(staticProps, remoteProps);
  const { getItem } = combinedProps;
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const handleFetchMore = useCallback(() => retrieve((p) => p), [retrieve]);
  const contextMenuProps = PMenu.useContextMenu();
  const canCreate = Access.useCreateGranted(view.TYPE_ONTOLOGY_ID);
  const renameModal = Modals.useRename();
  const { update: create } = PView.useCreate({
    beforeUpdate: async ({ data, rollbacks }) => {
      const name = await renameModal(
        { initialValue: `View for ${plural(resourceType)}` },
        { name: "View.Create" },
      );
      if (name == null) return false;
      const newKey = uuid.create();
      const previousSelected = selected;
      onSelect(newKey);
      rollbacks.push(() => onSelect(previousSelected));
      return { ...data, name, key: newKey };
    },
  });
  const handleCreate = () => {
    const currentQuery = getItem(selected)?.query;
    if (currentQuery == null) throw new UnexpectedError("No current query found");
    create({ name: `View for ${resourceType}`, type: resourceType, query: {} });
  };
  const canEditView = Access.useUpdateGranted(view.ontologyID(selected));
  return (
    <Select.Frame
      {...combinedProps}
      value={selected}
      onChange={onSelect}
      onFetchMore={handleFetchMore}
    >
      <Controls x>
        {canCreate && (
          <Button.Button
            onClick={handleCreate}
            tooltip="Create a view"
            size="small"
            tooltipLocation={location.BOTTOM_LEFT}
          >
            <Icon.Add />
          </Button.Button>
        )}
        {canEditView && (
          <Button.Toggle
            size="small"
            value={editable}
            onChange={() => setEditable((prev) => !prev)}
            tooltip={`${editable ? "Disable" : "Enable"} editing`}
            tooltipLocation={location.BOTTOM_LEFT}
          >
            {editable ? <Icon.EditOff /> : <Icon.Edit />}
          </Button.Toggle>
        )}
      </Controls>
      <PMenu.ContextMenu {...contextMenuProps} menu={contextMenu}>
        <List.Items {...combinedProps}>{item}</List.Items>
      </PMenu.ContextMenu>
    </Select.Frame>
  );
};

const ContextMenu = ({ keys }: PMenu.ContextMenuMenuProps): ReactElement | null => {
  const { getItem } = List.useUtilContext<view.Key, View>();
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const views = getItem(keys);
  const filteredViews = views.filter((v) => v.static !== true);
  const confirm = Ontology.useConfirmDelete({
    icon: "View",
    type: "Delete",
    description: "Deletion will permanently remove the view(s).",
  });
  const { update: del } = PView.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<PView.DeleteParams>) => {
        const views = getItem(keys);
        const confirmed = await confirm(views);
        if (!confirmed) return false;
        return data;
      },
      [getItem, confirm],
    ),
  });
  const canRename = filteredViews.length === 1;
  const canDelete = filteredViews.length > 0;
  return (
    <PMenu.Menu level="small" gap="small">
      {canRename && (
        <PMenu.Item itemKey="rename" onClick={() => Text.edit(filteredViews[0].key)}>
          <Icon.Rename />
          Rename
        </PMenu.Item>
      )}
      {canDelete && (
        <PMenu.Item
          itemKey="delete"
          onClick={() => del(filteredViews.map(({ key }) => key))}
        >
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

const Item = ({ itemKey }: List.ItemProps<view.Key>): ReactElement | null => {
  const item = List.useItem<view.Key, View>(itemKey);
  const { update: rename } = PView.useRename();
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
