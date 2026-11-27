// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/channel/Select.css";

import type { channel, Synnax as Client } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import type { ReactElement, ReactNode } from "react";

import { getCachedIndex, HaulPreview } from "@/channel/HaulPreview";
import { type ListQuery, useList } from "@/channel/queries";
import { resolveIcon } from "@/channel/resolveIcon";
import { HAUL_TYPE } from "@/channel/types";
import { CSS } from "@/css";
import type { Flux } from "@/flux";
import { Synnax } from "@/synnax";

/**
 * @returns the names of the folders on the channel's path, joined as the path joins
 * them, or undefined for a channel at the top or with no path.
 */
const folderOf = ({ name, path }: channel.Channel): string | undefined =>
  path == null || path === name
    ? undefined
    : path.slice(0, path.length - name.length - 1);

const FOLDER_CLASS = CSS.BE("channel", "folder");

/** Shows the folders of a channel's path, so two channels of one name differ. */
const Folder = ({ channel: ch }: { channel: channel.Channel }): ReactElement | null => {
  const folder = folderOf(ch);
  if (folder == null) return null;
  return (
    <Text.Text
      el="span"
      level="small"
      color={9}
      overflow="fade"
      className={FOLDER_CLASS}
    >
      {folder}
    </Text.Text>
  );
};

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<channel.Key>): ReactElement | null => {
  const item = List.useItem<channel.Key, channel.Channel>(itemKey);
  const Icon = resolveIcon(item?.payload);
  return (
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon />
      {item?.name}
      {item != null && <Folder channel={item} />}
    </Select.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectMultipleProps
  extends
    Omit<
      Select.MultipleProps<channel.Key, channel.Channel | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Omit<Flux.UseListParams<ListQuery, channel.Channel>, "name"> {}

const isChannel = (entry: unknown): entry is channel.Channel =>
  entry != null && typeof entry === "object" && "payload" in entry;

// A selection with no channel, such as a fixed item, shows the select's own icon.
const renderTriggerIcon = (entry: unknown): Icon.ReactElement | undefined => {
  if (!isChannel(entry)) return undefined;
  const Resolved = resolveIcon(entry.payload);
  return <Resolved />;
};

const renderTriggerFolder = (entry: unknown): ReactNode =>
  isChannel(entry) ? <Folder channel={entry} /> : undefined;

/** Lists the channels with their paths. */
const withPaths = (query: ListQuery | null | undefined): ListQuery | null =>
  query === null ? null : { ...query, includePath: true };

const createHaulPreviewRenderer =
  (client: Client | null) =>
  (entry: unknown): ReactNode =>
    isChannel(entry) ? (
      <HaulPreview
        channel={entry.payload}
        index={getCachedIndex(client, entry.payload)}
      />
    ) : undefined;

const TRIGGER_CLASS = CSS.BE("channel", "trigger");

// A caller's trigger props add to the defaults, so the trigger keeps its icon and mark.
const withDefaults = <P extends { className?: string }>(
  client: Client | null,
  props?: P,
) => ({
  renderIcon: renderTriggerIcon,
  renderHaulPreview: createHaulPreviewRenderer(client),
  ...props,
  className: CSS.cls(TRIGGER_CLASS, props?.className),
});

export const SelectMultiple = ({
  query,
  filter,
  triggerProps,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const client = Synnax.use();
  const { data, getItem, subscribe, status, more, search } = useList({
    query: withPaths(query),
    filter,
  });
  return (
    <Select.Multiple<channel.Key, channel.Channel | undefined>
      haulType={HAUL_TYPE}
      onFetchMore={more}
      onSearch={search}
      status={status}
      icon={<Icon.Channel />}
      triggerProps={withDefaults(client, triggerProps)}
      {...rest}
      resourceName="channel"
      data={data}
      getItem={getItem}
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<channel.Key, channel.Channel | undefined>,
      "data" | "getItem" | "subscribe" | "children" | "resourceName"
    >,
    Omit<Flux.UseListParams<ListQuery, channel.Channel>, "name"> {}

export const SelectSingle = ({
  query,
  filter,
  triggerProps,
  ...rest
}: SelectSingleProps): ReactElement => {
  const client = Synnax.use();
  const { data, getItem, subscribe, status, more, search } = useList({
    query: withPaths(query),
    filter,
  });
  return (
    <Select.Single<channel.Key, channel.Channel | undefined>
      onFetchMore={more}
      onSearch={search}
      status={status}
      haulType={HAUL_TYPE}
      icon={<Icon.Channel />}
      triggerProps={{
        renderEndContent: renderTriggerFolder,
        ...withDefaults(client, triggerProps),
      }}
      {...rest}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      resourceName="channel"
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
