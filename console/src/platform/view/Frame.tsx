// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/view/View.css";

import { type ontology, UnexpectedError, view } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  Flex,
  type Flux,
  Icon,
  List,
  Menu,
  Select,
  Tabs,
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

import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import {
  Context,
  type StaticView,
  useContext,
  type View,
} from "@/platform/view/context";
import { Vis } from "@/platform/vis";

export interface FrameProps extends PropsWithChildren {
  resourceType: ontology.ResourceType;
  icon: string;
}

export const Frame = ({ resourceType, icon, children }: FrameProps): ReactElement => {
  const staticViewKey = useMemo(() => uuid.create(), []);
  const staticViews = useMemo<StaticView[]>(
    () => [
      {
        key: staticViewKey,
        name: `All ${plural(resourceType)}`,
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
  const combinedProps = List.useCombinedData<view.Key, View>({
    first: staticProps,
    second: remoteProps,
  });
  const { getItem } = combinedProps;
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const staticViewKeys = useMemo(() => staticViews.map((v) => v.key), [staticViews]);
  const [selected, setSelected] = useState(staticViews[0].key);
  const hasUpdatePermission = Access.useUpdateGranted(view.ontologyID(selected));
  const [editable, setEditable] = useState(false);
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
      editable: editable && hasUpdatePermission,
      staticViews: staticViewKeys,
      select: setSelected,
      getInitialView,
    }),
    [
      resourceType,
      selected,
      editable,
      hasUpdatePermission,
      staticViewKeys,
      getInitialView,
    ],
  );

  return (
    <Flex.Box full="y" empty>
      <Context value={contextValue}>
        <Selector
          icon={icon}
          showEditButton={
            staticViewKeys.includes(selected) ? true : hasUpdatePermission
          }
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
    </Flex.Box>
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
  icon: string;
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
  icon,
}: SelectorProps): ReactElement => {
  const { getItem } = listProps;
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const contextMenuProps = Menu.useContextMenu();
  const hasCreatePermission = Access.useCreateGranted(view.TYPE_ONTOLOGY_ID);
  const renameModal = Modals.useRename();
  const { update: create } = PView.useCreate({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<view.New>) => {
        const name = await renameModal({
          initialValue: `View for ${plural(resourceType)}`,
          title: "View.Create",
          icon: Icon.resolve(icon),
        });
        if (name == null) return false;
        const newKey = uuid.create();
        const previousSelected = selected;
        rollbacks.push(() => onSelect(previousSelected));
        return { ...data, name, key: newKey };
      },
      [renameModal, resourceType, selected, icon],
    ),
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<view.New>) => {
        onSelect(data?.key ?? "");
      },
      [onSelect],
    ),
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
      <Vis.Controls x>
        {hasCreatePermission && editable && (
          <Button.Button
            onClick={handleCreate}
            tooltip="Create view"
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
      </Vis.Controls>
      <Menu.ContextMenu {...contextMenuProps} menu={contextMenu}>
        <Tabs.Frame className={CSS.BE("view", "views")}>
          <Strip onContextMenu={contextMenuProps.open} />
        </Tabs.Frame>
      </Menu.ContextMenu>
    </Select.Frame>
  );
};

const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
  const { selected, select, staticViews, resourceType } = useContext("View.Selector");
  const { getItem } = List.useUtilContext<view.Key, View>();
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const views = getItem(keys);
  const filteredViews = views.filter((v) => v.static !== true);
  const confirm = Modals.useConfirmDelete({
    icon: "View",
    type: caseconv.capitalize(resourceType),
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
  const canRename =
    Access.useUpdateGranted(view.ontologyID(filteredViews.map(({ key }) => key))) &&
    filteredViews.length === 1;
  const canDelete =
    Access.useDeleteGranted(view.ontologyID(filteredViews.map(({ key }) => key))) &&
    filteredViews.length > 0;
  return (
    <PlatformContextMenu.Menu>
      {canRename && (
        <PlatformContextMenu.RenameItem
          onClick={() => Text.edit(List.itemNameID(filteredViews[0].key))}
        />
      )}
      {canDelete && (
        <PlatformContextMenu.DeleteItem
          onClick={() => del(filteredViews.map(({ key }) => key))}
        />
      )}
      {(canRename || canDelete) && <Menu.Divider />}
      <PlatformContextMenu.ReloadConsoleItem />
    </PlatformContextMenu.Menu>
  );
};

const contextMenu = Component.renderProp(ContextMenu);

interface StripProps {
  onContextMenu: Menu.ContextMenuOpen;
}

/**
 * Strip lays the frame's views out as a tab strip. It stands in for List.Items, whose
 * scroll container the strip owns instead.
 */
const Strip = ({ onContextMenu }: StripProps): ReactElement => {
  const { data, ref, sentinelRef } = List.useData<view.Key, View>();
  return (
    <Tabs.Selector
      ref={ref}
      size="small"
      sizing="content"
      onContextMenu={onContextMenu}
    >
      {data.map((key) => (
        <Item key={key} itemKey={key} />
      ))}
      {sentinelRef != null && (
        <div ref={sentinelRef} className={CSS.BE("view", "sentinel")} aria-hidden />
      )}
    </Tabs.Selector>
  );
};

interface ItemProps {
  itemKey: view.Key;
}

const Item = ({ itemKey }: ItemProps): ReactElement | null => {
  const item = List.useItem<view.Key, View>(itemKey);
  const { update: rename } = PView.useRename();
  const canRename = Access.useUpdateGranted(view.ontologyID(itemKey));
  const handleRename = useCallback(
    (name: string) => rename({ key: itemKey, name }),
    [itemKey, rename],
  );
  if (item == null) return null;
  const { name } = item;
  return (
    <Tabs.Tab itemKey={itemKey}>
      <Text.MaybeEditable
        id={List.itemNameID(itemKey)}
        value={name}
        allowDoubleClick={false}
        onChange={canRename && item.static !== true ? handleRename : undefined}
      />
    </Tabs.Tab>
  );
};
