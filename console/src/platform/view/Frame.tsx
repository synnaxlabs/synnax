// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/view/View.css";

import { type panel, UnexpectedError, view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, Panel as PPanel, View as PView } from "@synnaxlabs/pluto";
import { caseconv, uuid } from "@synnaxlabs/x";
import { plural } from "pluralize";
import {
  Fragment,
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  useState,
} from "react";

import { Appearance } from "@/platform/appearance";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";
import {
  Context,
  type ContextValue,
  SearchContext,
  type SearchContextValue,
  type StaticView,
  useContext,
  type View,
} from "@/platform/view/context";
import { IconPicker, openIconPicker } from "@/platform/view/IconPicker";
import { Search } from "@/platform/view/Search";
import { useCreateModal } from "@/platform/view/useCreateModal";

export interface FrameProps extends PropsWithChildren {
  resourceType: view.Type;
  icon: string;
  /** The type of the explorer tab, which shows the static view. */
  tabType: string;
}

const ZERO_VARIANTS: {
  [K in view.Type]: Omit<Extract<view.View, { type: K }>, keyof view.Base>;
} = {
  range: { type: "range", sort: "newest", layout: "tree" },
  status: { type: "status" },
};

/**
 * Frame shows the views of a resource type above the open one. The panel stores the
 * view a tab shows, so everyone who opens the panel sees it. Must render in an explorer
 * tab of the given type, or in a tab of a saved view.
 */
export const Frame = (props: FrameProps): ReactElement => {
  const tab = PPanel.useTab({});
  if (tab.variant === "view") return <Body {...props} tab={tab.key} open={null} />;
  return <SavedBody {...props} tab={tab.key} openKey={tab.resource.key} />;
};

interface BodyProps extends FrameProps {
  tab: panel.TabKey;
  open: view.View | null;
}

interface SavedBodyProps extends Omit<BodyProps, "open"> {
  openKey: view.Key;
}

const SavedBody = ({ openKey, ...rest }: SavedBodyProps): ReactElement => {
  const open = PView.use({ key: openKey });
  return <Body {...rest} open={open} />;
};

const Body = ({
  resourceType,
  icon,
  tabType,
  tab,
  open,
  children,
}: BodyProps): ReactElement => {
  const openTab = Panel.useOpenTab();
  const canChange = Panel.useCanOpenTab();
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
  const selected = open?.key ?? staticViews[0].key;
  const tabOf = (key: view.Key): panel.NewTab =>
    staticViewKeys.includes(key)
      ? { variant: "view", type: tabType }
      : { variant: "resource", resource: view.resourceID(key) };
  const select = (key: view.Key) => {
    // A user who may not change the panel keeps the view it stores.
    if (!canChange) return;
    // A view open in another tab of the panel focuses that tab instead.
    openTab({ ...tabOf(key), key: tab });
    setSearchTerm("");
  };
  const openInNewTab = (key: view.Key) => openTab(tabOf(key));
  const updateGranted = Access.useUpdateGranted(view.resourceID(selected));
  // A static view saves nothing, so everyone may change it.
  const editable = staticViewKeys.includes(selected) || updateGranted;
  const getInitialView = () => {
    const view = open ?? getItem(selected);
    if (view == null) throw new UnexpectedError("No view found");
    return view;
  };
  const contextValue: ContextValue = {
    resourceType,
    selected,
    editable,
    staticViews: staticViewKeys,
    select,
    openInNewTab: canChange ? openInNewTab : null,
    getInitialView,
  };
  const searchValue: SearchContextValue = { term: searchTerm, setTerm: setSearchTerm };

  return (
    <Flex.Box full="y" empty>
      <Context value={contextValue}>
        <SearchContext value={searchValue}>
          <Selector
            icon={icon}
            resourceType={resourceType}
            onFetchMore={remoteProps.more}
            onSelect={select}
            listProps={combinedProps}
            selected={selected}
          />
          <Fragment key={selected}>{children}</Fragment>
        </SearchContext>
      </Context>
    </Flex.Box>
  );
};

interface SelectorProps {
  resourceType: view.Type;
  onSelect: (key: view.Key) => void;
  selected: view.Key;
  listProps: List.FrameProps<view.Key, View>;
  onFetchMore: () => void;
  icon: string;
}

const Selector = ({
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
  return (
    <Select.Frame
      {...listProps}
      value={selected}
      onChange={onSelect}
      onFetchMore={onFetchMore}
    >
      <Menu.ContextMenu {...contextMenuProps} menu={contextMenu}>
        <Tabs.Frame x align="center" className={CSS.BE("view", "views")}>
          <Search />
          <Strip
            onContextMenu={contextMenuProps.open}
            icon={icon}
            onAnchorChange={setAnchor}
          />
          {hasCreatePermission && (
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

// A panel shows a saved view in one tab at most, so the one this tab shows opens in no
// other.
const opensInNewTab = (
  { openInNewTab, selected, staticViews }: ContextValue,
  key: view.Key,
): boolean => openInNewTab != null && (key !== selected || staticViews.includes(key));

const ContextMenu = ({
  keys,
  target,
}: Menu.ContextMenuMenuProps): ReactElement | null => {
  const ctx = useContext("View.Selector");
  const { openInNewTab, staticViews, resourceType } = ctx;
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
      return data;
    },
  });
  const canRename =
    Access.useUpdateGranted(view.resourceID(editableKeys)) && editableKeys.length === 1;
  const canDelete =
    Access.useDeleteGranted(view.resourceID(editableKeys)) && editableKeys.length > 0;
  const canOpen = keys.length === 1 && opensInNewTab(ctx, keys[0]);
  return (
    <PlatformContextMenu.Menu>
      {canOpen && (
        <>
          <Menu.Item itemKey="open-in-new-tab" onClick={() => openInNewTab?.(keys[0])}>
            <Icon.OpenInNewWindow />
            Open in new tab
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {canRename && (
        <PlatformContextMenu.RenameItem
          onClick={() => Text.edit(List.itemNameID(editableKeys[0]), target)}
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
  onAnchorChange: Appearance.SetAnchor;
}

/**
 * Strip lays the frame's views out as a tab strip. It stands in for List.Items, whose
 * scroll container the strip owns instead.
 */
const Strip = ({ onContextMenu, icon, onAnchorChange }: StripProps): ReactElement => {
  const { data, ref, sentinelRef } = List.useData<view.Key, View>();
  return (
    <Tabs.Selector
      ref={ref}
      aria-label="Views"
      variant="text"
      size="medium"
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
  const ctx = useContext("View.Item");
  const { update: rename } = PView.useRename();
  const canRename = Access.useUpdateGranted(view.resourceID(itemKey));
  const handleRename = (name: string) => rename({ key: itemKey, name });
  const handleRef = (tab: HTMLDivElement | null) => onAnchorChange(itemKey, tab);
  // The capture runs before the tab selects itself.
  const handleClickCapture = (e: MouseEvent) => {
    if (!(e.ctrlKey || e.metaKey) || !opensInNewTab(ctx, itemKey)) return;
    e.stopPropagation();
    ctx.openInNewTab?.(itemKey);
  };
  if (item == null) return null;
  const { name } = item;
  return (
    <Tabs.Tab ref={handleRef} itemKey={itemKey} onClickCapture={handleClickCapture}>
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
