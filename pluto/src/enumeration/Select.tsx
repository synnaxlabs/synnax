// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/enumeration/Select.css";

import type { enumeration, Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Text } from "@synnaxlabs/lyra/text";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Colors } from "@/enumeration/Colors";
import { OptionMenu, useHasOptionMenu } from "@/enumeration/Edit";
import { Flux } from "@/flux";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<enumeration.Key>): ReactElement | null => {
  const item = List.useItem<enumeration.Key, enumeration.Enumeration>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item itemKey={itemKey} {...rest} align="center">
      <Icon.Enumeration />
      <Text.Text el="span" overflow="fade" grow>
        {item.name}
      </Text.Text>
      <Entries entries={item.entries} />
    </Select.Item>
  );
};

const ENTRIES_CLASS = CSS.BE("enumeration-select", "entries");
const ENTRY: Tag.TagsProps["noun"] = { one: "entry", other: "entries" };

interface EntriesProps {
  entries: enumeration.Entry[];
}

// An entry with no name has no text for a tag, so only named entries show.
const Entries = ({ entries }: EntriesProps): ReactElement => {
  const named = entries.filter(({ name }) => name.trim() !== "");
  const colors = Color.useValues(named.map(({ color }) => color));
  return (
    <Tag.Tags variant="text" unstacked noun={ENTRY} className={ENTRIES_CLASS}>
      {named.map(({ key, name }, i) => (
        <Tag.Tag key={key} color={colors[i]} size="small">
          {name}
        </Tag.Tag>
      ))}
    </Tag.Tags>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);
const optionMenuRenderProp = Component.renderProp(OptionMenu);

const renderColors = (entry: unknown): ReactElement | null =>
  entry == null ? null : (
    <Colors entries={(entry as enumeration.Enumeration).entries} />
  );

const DIALOG_CLASS = CSS.BE("enumeration-select", "dialog");
const TRIGGER_CLASS = CSS.BE("enumeration-select", "trigger");

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<enumeration.Key, enumeration.Enumeration | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Omit<
      Flux.UseListParams<enumeration.RetrieveMultipleParams, enumeration.Enumeration>,
      "name"
    > {}

export const SelectSingle = ({
  filter,
  query,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.enumerations,
    {
      name: "enumerations",
      query: query === null ? null : { order: { field: "name" }, ...query },
      filter,
    },
  );
  const hasMenu = useHasOptionMenu();
  return (
    <Select.Single<enumeration.Key, enumeration.Enumeration | undefined>
      getItem={getItem}
      onFetchMore={more}
      onSearch={search}
      status={status}
      icon={<Icon.Enumeration />}
      menu={optionMenuRenderProp}
      hasMenu={hasMenu}
      {...rest}
      triggerProps={{
        renderEndContent: renderColors,
        ...rest.triggerProps,
        className: CSS.cls(TRIGGER_CLASS, rest.triggerProps?.className),
      }}
      dialogProps={{
        ...rest.dialogProps,
        className: CSS.cls(DIALOG_CLASS, rest.dialogProps?.className),
      }}
      data={data}
      resourceName="enumeration"
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
