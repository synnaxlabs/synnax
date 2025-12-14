// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/View.css";

import { type ontology, UnexpectedError, view } from "@synnaxlabs/client";
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
import { caseconv, location, uuid } from "@synnaxlabs/x";
import { plural } from "pluralize";
import {
  Fragment,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { Controls, Menu } from "@/components";
import { CSS } from "@/css";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";
import { Context, type StaticView, useContext, type View } from "@/view/context";

export interface ProviderProps extends PropsWithChildren {
  resourceType: ontology.ResourceType;
}

export const Provider = ({ resourceType, children }: ProviderProps): ReactElement => {
  const staticViewKey = useMemo(() => uuid.create(), []);
  const staticViews = useMemo<StaticView[]>(
    () => [
      {
        key: staticViewKey,
        name: `All ${caseconv.capitalize(plural(resourceType))}`,
        type: resourceType,
        query: {},
        static: true,
      },
    ],
    [resourceType, staticViewKey],
  );
  const staticProps = List.useStaticData<view.Key, View>({ data: staticViews });
  const remoteProps = PView.useList({ initialQuery: { types: [resourceType] } });
  const { retrieve } = remoteProps;
  const handleFetchMore = useCallback(() => retrieve((p) => p), [retrieve]);
  const combinedProps = List.useCombinedData<view.Key, View>(staticProps, remoteProps);
  const { getItem } = combinedProps;
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const staticViewKeys = useMemo(() => staticViews.map((v) => v.key), [staticViews]);
  const [selected, setSelected] = useState(staticViews[0].key);
  const canUpdateView = Access.useUpdateGranted(view.ontologyID(selected));
  const [editable, setEditable] = useState(canUpdateView);
  const getInitialView = useCallback(() => {
    const view = getItem(selected);
    if (view == null) throw new UnexpectedError("No view found");
    return view;
  }, [getItem, selected]);
  const deleteSynchronizer = useCallback(
    (key: view.Key) => {
      if (key !== selected) return;
      setSelected(staticViews[0].key);
    },
    [selected, staticViews[0].key],
  );
  PView.useDeleteSynchronizer(deleteSynchronizer);
  const contextValue = useMemo(
    () => ({
      resourceType,
      selected,
      editable: editable && canUpdateView,
      staticViews: staticViewKeys,
      select: setSelected,
      getInitialView,
    }),
    [resourceType, selected, editable, canUpdateView, staticViewKeys, getInitialView],
  );

  return (
    <Context value={contextValue}>
      <Selector
        showEditButton={staticViewKeys.includes(selected) ? true : canUpdateView}
        editable={editable}
        onEditableClick={() => setEditable((prev) => !prev)}
        resourceType={resourceType}
        onFetchMore={handleFetchMore}
        staticViews={staticViews}
        onSelect={setSelected}
        listProps={combinedProps}
        selected={selected}
      />
      <Fragment key={selected}>{children}</Fragment>
    </Context>
  );
};

interface SelectorProps {
  showEditButton: boolean;
  editable: boolean;
  onEditableClick: () => void;
  resourceType: ontology.ResourceType;
  staticViews: StaticView[];
  onSelect: (key: view.Key) => void;
  selected: view.Key;
  listProps: List.FrameProps<view.Key, View>;
  onFetchMore: () => void;
}

const Selector = ({
  showEditButton,
  editable,
  onEditableClick,
  resourceType,
  onFetchMore,
  onSelect,
  listProps,
  selected,
}: SelectorProps): ReactElement => {
  const { getItem } = listProps;
  if (getItem == null) throw new UnexpectedError("No item getter found");
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
      rollbacks.push(() => onSelect(previousSelected));
      return { ...data, name, key: newKey };
    },
    afterSuccess: ({ data }) => {
      onSelect(data?.key ?? "");
    },
  });
  const handleCreate = () => {
    const currentQuery = getItem(selected)?.query;
    if (currentQuery == null) throw new UnexpectedError("No current query found");
    create({ name: `View for ${resourceType}`, type: resourceType, query: {} });
  };
  return (
    <Select.Frame
      {...listProps}
      value={selected}
      onChange={onSelect}
      onFetchMore={onFetchMore}
    >
      <Controls x>
        {canCreate && editable && (
          <Button.Button
            onClick={handleCreate}
            tooltip="Create a view"
            size="small"
            tooltipLocation={location.BOTTOM_LEFT}
          >
            <Icon.Add />
          </Button.Button>
        )}
        {showEditButton && (
          <Button.Toggle
            size="small"
            value={editable}
            onChange={onEditableClick}
            tooltip={`${editable ? "Disable" : "Enable"} editing`}
            tooltipLocation={location.BOTTOM_LEFT}
          >
            {editable ? <Icon.EditOff /> : <Icon.Edit />}
          </Button.Toggle>
        )}
      </Controls>
      <PMenu.ContextMenu {...contextMenuProps} menu={contextMenu}>
        <List.Items
          className={CSS.BE("view", "views")}
          x
          align="center"
          gap="medium"
          onContextMenu={contextMenuProps.open}
        >
          {item}
        </List.Items>
      </PMenu.ContextMenu>
    </Select.Frame>
  );
};

const ContextMenu = ({ keys }: PMenu.ContextMenuMenuProps): ReactElement | null => {
  const { selected, select, staticViews, resourceType } = useContext("View.Selector");
  const { getItem } = List.useUtilContext<view.Key, View>();
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const views = getItem(keys);
  const filteredViews = views.filter((v) => v.static !== true);
  const confirm = Ontology.useConfirmDelete({
    icon: "View",
    type: caseconv.capitalize(plural(resourceType)),
    description: "Deletion will permanently remove the view(s).",
  });
  const { update: del } = PView.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<PView.DeleteParams>) => {
        const views = getItem(keys);
        const confirmed = await confirm(views);
        if (!confirmed) return false;
        if (keys.includes(selected)) select(staticViews[0]);
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
