// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/view/View.css";

import { UnexpectedError, view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, View as PView } from "@synnaxlabs/pluto";
import { caseconv, location, uuid } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { Fragment, type PropsWithChildren, type ReactElement, useState } from "react";

import { Appearance } from "@/platform/appearance";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import {
  Context,
  type ContextValue,
  type StaticView,
  useContext,
  type View,
} from "@/platform/view/context";
import { IconPicker, openIconPicker } from "@/platform/view/IconPicker";
import { SearchInput } from "@/platform/view/Search";
import { useCreateModal } from "@/platform/view/useCreateModal";
import { Vis } from "@/platform/vis";

export interface FrameProps extends PropsWithChildren {
  resourceType: view.Type;
  icon: string;
  /** Keeps the search and the filters open, with no toggle to enable editing. */
  alwaysEditable?: boolean;
  /** Leads the views bar with the search, over a bar that holds the filters. */
  searchInViews?: boolean;
}

const ZERO_VARIANTS: {
  [K in view.Type]: Omit<Extract<view.View, { type: K }>, keyof view.Base>;
} = {
  range: { type: "range", sort: "newest", layout: "tree" },
  status: { type: "status" },
};

export const Frame = ({
  resourceType,
  icon,
  alwaysEditable = false,
  searchInViews = false,
  children,
}: FrameProps): ReactElement => {
  const [searchTerm, setSearchTerm] = useState("");
  const [staticViewKey] = useState(() => uuid.create());
  const staticViews: StaticView[] = [
    {
      key: staticViewKey,
      name: `All ${plural(resourceType)}`,
      ...ZERO_VARIANTS[resourceType],
      static: true,
    },
  ];
  const staticProps = List.useStaticData<view.Key, View>({ data: staticViews });
  const remoteProps = Flux.useList((client) => client.views, {
    name: "views",
    query: { where: { type: { eq: resourceType } } },
  });
  const combinedProps = List.useCombinedData<view.Key, View>({
    first: staticProps,
    second: remoteProps,
  });
  const { getItem } = combinedProps;
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const staticViewKeys = staticViews.map((v) => v.key);
  const [selected, setSelected] = useState(staticViews[0].key);
  const select = (key: view.Key) => {
    setSelected(key);
    setSearchTerm("");
  };
  const updateGranted = Access.useUpdateGranted(view.resourceID(selected));
  // A static view saves nothing, so everyone may change it.
  const updatable = staticViewKeys.includes(selected) || updateGranted;
  const [editable, setEditable] = useState(false);
  const getInitialView = () => {
    const view = getItem(selected);
    if (view == null) throw new UnexpectedError("No view found");
    return view;
  };
  const deleteSynchronizer = (key: view.Key) => {
    if (key !== selected) return;
    select(staticViews[0].key);
  };
  PView.useDeleteSynchronizer(deleteSynchronizer);
  const editing = alwaysEditable || editable;
  const contextValue: ContextValue = {
    resourceType,
    selected,
    editing,
    editable: editing && updatable,
    staticViews: staticViewKeys,
    select,
    getInitialView,
    searchInViews,
    searchTerm,
    setSearchTerm,
  };

  return (
    <Flex.Box full="y" empty>
      <Context value={contextValue}>
        <Selector
          icon={icon}
          showEditButton={!alwaysEditable}
          editable={editing}
          onEditableClick={() => setEditable((prev) => !prev)}
          resourceType={resourceType}
          onFetchMore={remoteProps.more}
          onSelect={select}
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
  resourceType: view.Type;
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
  const contextMenuProps = Menu.useContextMenu();
  const hasCreatePermission = Access.useCreateGranted(view.TYPE_RESOURCE_ID);
  const createModal = useCreateModal();
  const handleCreate = async () => {
    const initialValues = { ...ZERO_VARIANTS[resourceType], name: "" };
    const result = await createModal({ initialValues });
    if (result != null) onSelect(result.key);
  };
  const [anchors, setAnchor] = Appearance.useAnchors();
  const { searchInViews } = useContext("View.Selector");
  const creatable = hasCreatePermission && editable;
  return (
    <Select.Frame
      {...listProps}
      value={selected}
      onChange={onSelect}
      onFetchMore={onFetchMore}
    >
      <Vis.Controls x>
        {creatable && !searchInViews && (
          <Button.Button
            onClick={() => void handleCreate()}
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
        <Tabs.Frame
          x={searchInViews}
          align={searchInViews ? "center" : undefined}
          className={CSS.cls(
            CSS.BE("view", "views"),
            searchInViews && CSS.BEM("view", "views", "island"),
          )}
        >
          {searchInViews && (
            <div className={CSS.BE("view", "search")}>
              <SearchInput />
            </div>
          )}
          <Strip
            onContextMenu={contextMenuProps.open}
            icon={icon}
            pills={searchInViews}
            onAnchorChange={setAnchor}
          />
          {creatable && searchInViews && (
            <Button.Button
              onClick={() => void handleCreate()}
              variant="text"
              size="small"
              textColor={9}
              tooltip="Create view"
              tooltipLocation="bottom"
              className={CSS.BE("view", "create")}
            >
              <Icon.Add />
            </Button.Button>
          )}
        </Tabs.Frame>
      </Menu.ContextMenu>
      {Object.entries(anchors).map(([key, anchor]) => (
        <IconPicker key={key} viewKey={key} anchor={anchor} />
      ))}
    </Select.Frame>
  );
};

const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
  const { selected, select, staticViews, resourceType } = useContext("View.Selector");
  const { getItem } = List.useUtilContext<view.Key, View>();
  if (getItem == null) throw new UnexpectedError("No item getter found");
  const editableKeys = keys.filter((key) => !staticViews.includes(key));
  const confirm = Modals.useConfirmDelete({
    icon: "View",
    type: caseconv.capitalize(resourceType),
  });
  const { update: del } = PView.useDelete({
    beforeUpdate: async ({ data }: Flux.BeforeUpdateParams<PView.DeleteParams>) => {
      const views = getItem(keys);
      const confirmed = await confirm(views);
      if (!confirmed) return false;
      if (keys.includes(selected)) select(staticViews[0]);
      return data;
    },
  });
  const canRename =
    Access.useUpdateGranted(view.resourceID(editableKeys)) && editableKeys.length === 1;
  const canDelete =
    Access.useDeleteGranted(view.resourceID(editableKeys)) && editableKeys.length > 0;
  return (
    <PlatformContextMenu.Menu>
      {canRename && (
        <PlatformContextMenu.RenameItem
          onClick={() => Text.edit(List.itemNameID(editableKeys[0]))}
        />
      )}
      {canRename && (
        <Menu.Item
          itemKey="change-icon"
          onClick={() => openIconPicker(editableKeys[0])}
        >
          <Icon.FillColor />
          Change icon
        </Menu.Item>
      )}
      {canDelete && (
        <PlatformContextMenu.DeleteItem onClick={() => del(editableKeys)} />
      )}
      {(canRename || canDelete) && <Menu.Divider />}
      <PlatformContextMenu.ReloadConsoleItem />
    </PlatformContextMenu.Menu>
  );
};

const contextMenu = Component.renderProp(ContextMenu);

interface StripProps {
  onContextMenu: Menu.ContextMenuOpen;
  /** The icon of the views that show every resource. */
  icon: string;
  /** Draws the views as pills as tall as the controls beside them. */
  pills: boolean;
  onAnchorChange: Appearance.SetAnchor;
}

/**
 * Strip lays the frame's views out as a tab strip. It stands in for List.Items, whose
 * scroll container the strip owns instead.
 */
const Strip = ({
  onContextMenu,
  icon,
  pills,
  onAnchorChange,
}: StripProps): ReactElement => {
  const { data, ref, sentinelRef } = List.useData<view.Key, View>();
  return (
    <Tabs.Selector
      ref={ref}
      aria-label="Views"
      variant={pills ? "text" : "default"}
      size={pills ? "medium" : "small"}
      sizing="content"
      onContextMenu={onContextMenu}
    >
      {data.map((key) => (
        <Item key={key} itemKey={key} icon={icon} onAnchorChange={onAnchorChange} />
      ))}
      {sentinelRef != null && (
        <div ref={sentinelRef} className={CSS.BE("view", "sentinel")} aria-hidden />
      )}
    </Tabs.Selector>
  );
};

interface ItemProps extends Pick<StripProps, "icon" | "onAnchorChange"> {
  itemKey: view.Key;
}

const Item = ({ itemKey, icon, onAnchorChange }: ItemProps): ReactElement | null => {
  const item = List.useItem<view.Key, View>(itemKey);
  const { update: rename } = PView.useRename();
  const canRename = Access.useUpdateGranted(view.resourceID(itemKey));
  const handleRename = (name: string) => rename({ key: itemKey, name });
  const handleRef = (tab: HTMLDivElement | null) => onAnchorChange(itemKey, tab);
  if (item == null) return null;
  const { name } = item;
  return (
    <Tabs.Tab ref={handleRef} itemKey={itemKey}>
      {item.static === true
        ? Icon.resolve(icon)
        : Appearance.renderIcon({
            icon: item.icon,
            color: item.color,
            fallback: Icon.View,
          })}
      <Text.MaybeEditable
        id={List.itemNameID(itemKey)}
        value={name}
        allowDoubleClick={false}
        overflow="fade"
        onChange={canRename && item.static !== true ? handleRename : undefined}
      />
    </Tabs.Tab>
  );
};
