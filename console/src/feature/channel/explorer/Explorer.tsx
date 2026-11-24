// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/channel/explorer/Explorer.css";

import { channel, type group, view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Channel, type Flux, Group as PGroup, Telem } from "@synnaxlabs/pluto";
import { deep, location } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { contextMenu } from "@/feature/channel/explorer/ContextMenu";
import { TAB_TYPE } from "@/feature/channel/explorer/external";
import { Filter } from "@/feature/channel/explorer/filter";
import { Open } from "@/feature/channel/open";
import { listTreeRoot } from "@/feature/channel/tree";
import { Channel as PlatformChannel } from "@/platform/channel";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Group } from "@/platform/group";
import type { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { View } from "@/platform/view";

const toHit = (ch: channel.Channel) => Group.toHit("channel")(ch.payload);

const useMatches: Tree.UseMatches<channel.Where> = (query) => [
  Tree.toMatches(
    Channel.useResultList(
      query == null
        ? null
        : {
            searchTerm: query.term === "" ? undefined : query.term,
            where: query.filter,
            limit: query.limit,
          },
    ),
    toHit,
  ),
];

const useSearch = Tree.createSearch<channel.Where>({
  noun: "channels",
  useMatches,
  usePlacement: Group.usePlacement,
});

const wherePath = deep.path<{ where?: channel.Where }>().where;

/**
 * The empty state of both layouts. A filter offers to clear it. Otherwise, it offers to
 * create a channel.
 */
const NoChannels = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const canCreate = Access.useCreateGranted(channel.TYPE_RESOURCE_ID);
  const { editable } = View.useContext();
  const where = Form.useFieldValue<channel.Where | undefined>(wherePath, {
    optional: true,
  });
  const { set } = Form.useContext();
  const { term } = View.useSearch();
  if (where != null)
    return (
      <Empty.Action
        message="No channels match the filter"
        action={editable ? "Clear filters" : undefined}
        onClick={() => set(wherePath, undefined)}
      />
    );
  return (
    <Empty.Action
      message={term === "" ? "No channels" : "No channels match"}
      action={canCreate ? "Create channel" : undefined}
      onClick={() => openCreate()}
    />
  );
};

const NO_CHANNELS = <NoChannels />;

interface FoldersProps {
  where: channel.Where | undefined;
  term: string;
}

/**
 * Shows the channels under their folders, as the sidebar does. A filter or a search
 * shows only the matches, each under the folders that hold it.
 */
const Folders = ({ where, term }: FoldersProps): ReactElement => {
  const search = useSearch(channel.TYPE_RESOURCE_ID, {
    term,
    filter: where,
    layout: "groups",
  });
  return (
    <Flex.Box y grow empty className={CSS.BE("channel-explorer", "folders")}>
      <Tree.Tree
        root={channel.TYPE_RESOURCE_ID}
        listRoot={listTreeRoot}
        emptyContent={NO_CHANNELS}
        emptyMatches={NO_CHANNELS}
        search={search}
      />
    </Flex.Box>
  );
};

const ITEM_HEIGHT = 44;

/** @returns the names of the folders from the top down to the folder with the key. */
const folderNames = (key: group.Key, folders: group.Group[]): string[] => {
  const byKey = new Map(folders.map((f) => [f.key, f]));
  const names: string[] = [];
  let folder = byKey.get(key);
  while (folder != null) {
    names.unshift(folder.name);
    const parent = folder.parent;
    folder = parent?.type === "group" ? byKey.get(parent.key) : undefined;
  }
  return names;
};

interface FolderPathProps {
  folder: group.Key;
}

/** Shows the folders that hold a channel, and follows a rename or move of each. */
const FolderPath = ({ folder }: FolderPathProps): ReactElement | null => {
  const folders = PGroup.useResultAncestors({ keys: [folder] }).data;
  if (folders == null) return null;
  return (
    <Text.Text level="small" color={9} overflow="fade">
      {folderNames(folder, folders).join(" / ")}
    </Text.Text>
  );
};

const ListItem = (props: List.ItemProps<channel.Key>): ReactElement | null => {
  const { itemKey } = props;
  const ch = List.useItem<channel.Key, channel.Channel>(itemKey);
  const canRename = Access.useUpdateGranted(channel.resourceID(itemKey));
  const { update: rename } = Channel.useRename();
  const open = Open.use();
  const { term } = View.useSearch();
  if (ch == null) return null;
  const KindIcon = Channel.resolveIcon(ch.payload);
  const handleRename =
    canRename && !ch.internal
      ? (name: string) => rename({ key: itemKey, name })
      : undefined;
  return (
    <Select.Item<channel.Key>
      {...props}
      role="listitem"
      aria-label={ch.name}
      checkable
      justify="between"
      className={CSS.BE("channel-explorer", "item")}
      onActivate={() =>
        open({
          key: channel.resourceID(itemKey).key,
          id: channel.resourceID(itemKey),
          name: ch.name,
        })
      }
    >
      <KindIcon color={10} />
      <Flex.Box y empty grow className={CSS.BE("channel-explorer", "summary")}>
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          value={ch.name}
          onChange={handleRename}
          allowDoubleClick={false}
          overflow="fade"
          level="p"
          weight={450}
          highlight={term === "" ? undefined : term}
        />
        {ch.group != null && <FolderPath folder={ch.group} />}
      </Flex.Box>
      <Flex.Box x align="center" className={CSS.BE("channel-explorer", "meta")}>
        {ch.virtual && (
          <Tooltip.Dialog location="left">
            <Text.Text level="small">Virtual</Text.Text>
            <Icon.Virtual color={9} aria-label="Virtual" />
          </Tooltip.Dialog>
        )}
        <Text.Text
          level="small"
          color={9}
          className={CSS.BE("channel-explorer", "data-type")}
        >
          {Telem.resolveDataTypeName(ch.dataType)}
        </Text.Text>
      </Flex.Box>
    </Select.Item>
  );
};

const listItem = Component.renderProp(ListItem);

interface ListLayoutProps {
  list: Flux.UseListReturn<channel.Key, channel.Channel>;
}

const ListLayout = ({ list }: ListLayoutProps): ReactElement => {
  if (list.variant === "error")
    return (
      <Status.Summary
        className={CSS.BE("channel-explorer", "message")}
        variant="error"
        message="Failed to list channels"
        description={list.status.description}
      />
    );
  return (
    <View.Items<channel.Key>
      contextMenu={contextMenu}
      emptyContent={NO_CHANNELS}
      className={CSS.BE("channel-explorer", "list")}
    >
      {listItem}
    </View.Items>
  );
};

interface LayoutInfo {
  name: string;
  icon: Icon.ReactElement;
}

const LAYOUTS: Record<view.ChannelLayout, LayoutInfo> = {
  folders: { name: "Folders", icon: <Icon.Group /> },
  list: { name: "List", icon: <Icon.Menu /> },
};

const LAYOUT_CLASS = CSS.BE("channel-explorer", "layout");
const LAYOUT_NAME_CLASS = CSS.BE("channel-explorer", "layout-name");

interface LayoutSelectProps {
  value: view.ChannelLayout;
  onChange: (value: view.ChannelLayout) => void;
  preview: boolean;
}

const LayoutSelect = ({
  value,
  onChange,
  preview,
}: LayoutSelectProps): ReactElement => (
  <Select.Buttons<view.ChannelLayout>
    size="medium"
    value={value}
    onChange={onChange}
    preview={preview}
  >
    {view.CHANNEL_LAYOUTS.map((layout) => (
      <Select.Item<view.ChannelLayout>
        key={layout}
        itemKey={layout}
        size="medium"
        className={LAYOUT_CLASS}
        aria-label={LAYOUTS[layout].name}
        tooltip={LAYOUTS[layout].name}
        tooltipLocation="bottom"
        hideTooltip={View.textShown(LAYOUT_NAME_CLASS)}
      >
        {LAYOUTS[layout].icon}
        <span className={LAYOUT_NAME_CLASS}>{LAYOUTS[layout].name}</span>
      </Select.Item>
    ))}
  </Select.Buttons>
);

const Internal = (): ReactElement => {
  const { editable } = View.useContext();
  const { term } = View.useSearch();
  const initial = View.useInitialView("channel");
  const [where, setWhere] = useState(initial.where);
  const [layout, setLayout] = View.useField("channel", "layout");
  const list = Channel.useList({ query: layout === "list" ? { where } : null });
  const openCreate = PlatformChannel.useCreateModal();
  const openCalculated = PlatformChannel.useCalculatedModal();
  const canCreate = Access.useCreateGranted(channel.TYPE_RESOURCE_ID);
  return (
    <View.Form {...list} onWhereChange={setWhere} itemHeight={ITEM_HEIGHT}>
      <View.Toolbar properties={Filter.PROPERTIES}>
        <LayoutSelect value={layout} onChange={setLayout} preview={!editable} />
        {canCreate && (
          <>
            <Button.Button
              variant="outlined"
              size="medium"
              tooltip="Create calculated channel"
              tooltipLocation={location.BOTTOM_LEFT}
              onClick={() => openCalculated()}
            >
              <Channel.CreateCalculatedIcon />
            </Button.Button>
            <Button.Button
              variant="outlined"
              size="medium"
              tooltip="Create channel"
              tooltipLocation={location.BOTTOM_LEFT}
              onClick={() => openCreate()}
            >
              <Icon.Add />
            </Button.Button>
          </>
        )}
      </View.Toolbar>
      {layout === "folders" ? (
        <Folders where={where} term={term} />
      ) : (
        <ListLayout list={list} />
      )}
    </View.Form>
  );
};

export const Explorer: Panel.Content = () => (
  <View.Frame resourceType="channel" icon="Channel" tabType={TAB_TYPE}>
    <Internal />
  </View.Frame>
);
