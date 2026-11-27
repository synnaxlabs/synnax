// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/group/Select.css";

import type { group, Synnax as Client } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useSyncExternalStore } from "react";

import { CSS } from "@/css";
import { Flux } from "@/flux";
import { useResultAncestors } from "@/group/queries";
import { Synnax } from "@/synnax";

/** Joins the names on a path. Matches the separator of the Core. */
const SEPARATOR = ".";

/**
 * @returns the names of the groups above the group, from the top, joined as a path
 * joins them, or undefined for a group at the top or one whose groups above are not
 * cached.
 */
const parentsOf = (client: Client | null, key: group.Key): string | undefined => {
  const ancestors = client?.groups.getCachedAncestors(key);
  if (ancestors == null || ancestors.length === 0) return undefined;
  return ancestors
    .map(({ name }) => name)
    .reverse()
    .join(SEPARATOR);
};

/** Shows the groups above a group, so two groups of one name differ. */
const Parents = ({ itemKey }: { itemKey: group.Key }): ReactElement | null => {
  const client = Synnax.use();
  const subscribe = (onChange: () => void) =>
    client?.groups.onAncestorsChange(itemKey, onChange) ?? (() => {});
  const parents = useSyncExternalStore(subscribe, () => parentsOf(client, itemKey));
  if (parents == null) return null;
  return (
    <Text.Text
      level="small"
      color={9}
      overflow="fade"
      className={CSS.BE("group", "parents")}
    >
      {parents}
    </Text.Text>
  );
};

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<group.Key>): ReactElement | null => {
  const item = List.useItem<group.Key, group.Group>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon.Group />
      <Text.Text el="span" overflow="fade" className={CSS.BE("group", "name")}>
        {item.name}
      </Text.Text>
      <Parents itemKey={itemKey} />
    </Select.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

const isGroup = (entry: unknown): entry is group.Group =>
  entry != null && typeof entry === "object" && "scope" in entry;

const renderTriggerParents = (entry: unknown): ReactNode =>
  isGroup(entry) ? <Parents itemKey={entry.key} /> : undefined;

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<group.Key, group.Group | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Partial<
      Pick<Select.SingleProps<group.Key, group.Group | undefined>, "resourceName">
    >,
    Omit<Flux.UseListParams<group.RetrieveMultipleParams, group.Group>, "name"> {}

/**
 * Selects one group. Each listed group and the selected one show the names of the
 * groups above them.
 */
export const SelectSingle = ({
  filter,
  query,
  triggerProps,
  value,
  resourceName = "group",
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Client) => client.groups,
    { name: "groups", query, filter },
  );
  // Reads the groups above each listed group and the selected one, which name them.
  useResultAncestors({
    keys: primitive.isZero(value) || data.includes(value) ? data : [...data, value],
  });
  return (
    <Select.Single<group.Key, group.Group | undefined>
      getItem={getItem}
      onFetchMore={more}
      onSearch={search}
      status={status}
      triggerProps={{
        renderEndContent: renderTriggerParents,
        ...triggerProps,
        className: CSS.cls(CSS.BE("group", "trigger"), triggerProps?.className),
      }}
      value={value}
      {...rest}
      data={data}
      resourceName={resourceName}
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
