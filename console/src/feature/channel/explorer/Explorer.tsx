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
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel, Group as PGroup, Telem } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { TAB_TYPE } from "@/feature/channel/explorer/external";
import { Filter } from "@/feature/channel/explorer/filter";
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

const NoChannels = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const canCreate = Access.useCreateGranted(channel.TYPE_RESOURCE_ID);
  return (
    <Empty.Action
      message="No channels match"
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
    <Text.Text level="small" color={8} overflow="fade">
      {folderNames(folder, folders).join(" / ")}
    </Text.Text>
  );
};

const ListItem = (props: List.ItemProps<channel.Key>): ReactElement | null => {
  const ch = List.useItem<channel.Key, channel.Channel>(props.itemKey);
  if (ch == null) return null;
  const DataTypeIcon = Telem.resolveDataTypeIcon(ch.dataType);
  return (
    <Select.Item<channel.Key>
      {...props}
      checkable
      justify="between"
      className={CSS.BE("channel-explorer", "item")}
    >
      <Flex.Box y empty grow className={CSS.BE("channel-explorer", "summary")}>
        <Text.Text level="p" weight={450} overflow="fade">
          {ch.name}
        </Text.Text>
        {ch.group != null && <FolderPath folder={ch.group} />}
      </Flex.Box>
      <Flex.Box x align="center" className={CSS.BE("channel-explorer", "meta")}>
        {ch.isIndex && (
          <Text.Text level="small" color={9}>
            Index
          </Text.Text>
        )}
        {ch.virtual && !ch.isCalculated && <Icon.Virtual aria-label="Virtual" />}
        {ch.isCalculated && <Icon.Calculation aria-label="Calculated" />}
        <Text.Text level="small" color={9}>
          {DataTypeIcon != null && <DataTypeIcon />}
          {Telem.resolveDataTypeName(ch.dataType)}
        </Text.Text>
      </Flex.Box>
    </Select.Item>
  );
};

const listItem = Component.renderProp(ListItem);

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
        <View.Items<channel.Key>>{listItem}</View.Items>
      )}
    </View.Form>
  );
};

export const Explorer: Panel.Content = () => (
  <View.Frame resourceType="channel" icon="Channel" tabType={TAB_TYPE}>
    <Internal />
  </View.Frame>
);
