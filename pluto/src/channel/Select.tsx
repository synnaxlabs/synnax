// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/channel/Select.css";

import { type channel } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { type ListQuery, useList } from "@/channel/queries";
import { resolveIcon } from "@/channel/resolveIcon";
import { HAUL_TYPE } from "@/channel/types";
import { CSS } from "@/css";
import { type Flux } from "@/flux";

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
    Flux.UseListParams<ListQuery, channel.Key, channel.Channel> {}

const isChannel = (entry: unknown): entry is channel.Channel =>
  entry != null && typeof entry === "object" && "payload" in entry;

const renderTriggerIcon = (entry: unknown): Icon.ReactElement | undefined => {
  const Resolved = resolveIcon(isChannel(entry) ? entry.payload : undefined);
  return <Resolved />;
};

const TRIGGER_CLASS = CSS.BE("channel", "trigger");

// A caller's trigger props add to the defaults, so the trigger keeps its icon and mark.
const withDefaults = <P extends { className?: string }>(props?: P) => ({
  renderIcon: renderTriggerIcon,
  ...props,
  className: CSS.cls(TRIGGER_CLASS, props?.className),
});

export const SelectMultiple = ({
  initialQuery,
  filter,
  triggerProps,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    initialQuery,
    filter,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<channel.Key, channel.Channel | undefined>
      haulType={HAUL_TYPE}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      icon={<Icon.Channel />}
      triggerProps={withDefaults(triggerProps)}
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
    Flux.UseListParams<ListQuery, channel.Key, channel.Channel> {}

export const SelectSingle = ({
  initialQuery,
  filter,
  triggerProps,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    initialQuery,
    filter,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<channel.Key, channel.Channel | undefined>
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      haulType={HAUL_TYPE}
      icon={<Icon.Channel />}
      triggerProps={withDefaults(triggerProps)}
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
