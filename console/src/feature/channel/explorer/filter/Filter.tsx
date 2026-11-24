// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, group } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux, Group, Telem } from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { View } from "@/platform/view";

/** @returns one or two names, or else a count of the noun. */
const summarize = (names: string[], noun: string): string =>
  names.length > 0 && names.length <= 2 ? names.join(", ") : `${names.length} ${noun}`;

const AnyOf = <V,>({ value }: View.Filter.ConditionProps<V[]>): ReactElement => (
  <View.Filter.ChipOperator>
    {value.length === 1 ? "is" : "is any of"}
  </View.Filter.ChipOperator>
);

const Is = (): ReactElement => <View.Filter.ChipOperator>is</View.Filter.ChipOperator>;

const FolderItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<group.Key>): ReactElement | null => {
  const folder = List.useItem<group.Key, group.Group>(itemKey);
  const above = Group.useResultAncestors({ keys: [itemKey] }).data;
  if (folder == null) return null;
  const parents = folderParents(itemKey, above ?? []);
  return (
    <Select.Item<group.Key> itemKey={itemKey} {...rest}>
      <Icon.Group />
      {folder.name}
      {parents.length > 0 && (
        <Text.Text level="small" color={9} overflow="fade">
          {parents.join(" / ")}
        </Text.Text>
      )}
    </Select.Item>
  );
};

/** @returns the names of the folders above the folder with the key, from the top. */
const folderParents = (key: group.Key, folders: group.Group[]): string[] => {
  const byKey = new Map(folders.map((f) => [f.key, f]));
  const names: string[] = [];
  let parent = byKey.get(key)?.parent;
  while (parent?.type === "group") {
    const folder = byKey.get(parent.key);
    if (folder == null) break;
    names.unshift(folder.name);
    parent = folder.parent;
  }
  return names;
};

const folderItem = Component.renderProp(FolderItem);

const FolderPicker = ({
  value,
  onChange,
}: View.Filter.PickerProps<group.Key[]>): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client) => client.groups,
    { name: "folders", query: { scope: "channel", order: { field: "name" } } },
  );
  return (
    <Select.Frame<group.Key, group.Group | undefined>
      multiple
      value={value ?? []}
      onChange={(next: group.Key[]) => onChange(next.length === 0 ? null : next)}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
    >
      <Select.Dialog surface="menu" aria-label="Folders">
        <Select.Body<group.Key> resourceName="folder" onSearch={search} status={status}>
          {folderItem}
        </Select.Body>
      </Select.Dialog>
    </Select.Frame>
  );
};

const FolderValue = ({
  value,
}: View.Filter.ConditionProps<group.Key[]>): ReactElement => {
  // A filter can keep the key of a deleted folder, so count the folders that resolve.
  const { data: folders = [] } = Group.useResultList({ where: { key: { in: value } } });
  return (
    <>
      {summarize(
        folders.map(({ name }) => name),
        "folders",
      )}
    </>
  );
};

const FOLDER: View.Filter.Property<channel.Where, group.Key[]> = {
  name: "Folder",
  icon: Icon.Group,
  decode: (node) =>
    View.Filter.soleField(node) === "groups" &&
    node.groups != null &&
    node.groups.length > 0
      ? node.groups
      : null,
  encode: (groups) => ({ groups }),
  Picker: FolderPicker,
  Operator: AnyOf,
  Value: FolderValue,
};

const DATA_TYPES = DataType.ALL.filter((d) => !d.equals(DataType.UNKNOWN));

const nameOf = (dataType: string): string =>
  Telem.resolveDataTypeName(new DataType(dataType));

const DataTypePicker = ({
  value,
  onChange,
}: View.Filter.PickerProps<string[]>): ReactElement => (
  <Select.Frame<string, undefined>
    multiple
    value={value ?? []}
    onChange={(next: string[]) => onChange(next.length === 0 ? null : next)}
  >
    <Select.Dialog surface="menu" aria-label="Data types">
      <Select.List>
        {DATA_TYPES.map((d) => {
          const DataTypeIcon = Telem.resolveDataTypeIcon(d);
          return (
            <Select.Item<string> key={d.toString()} itemKey={d.toString()}>
              {DataTypeIcon != null && <DataTypeIcon />}
              {Telem.resolveDataTypeName(d)}
            </Select.Item>
          );
        })}
      </Select.List>
    </Select.Dialog>
  </Select.Frame>
);

const DATA_TYPE: View.Filter.Property<channel.Where, string[]> = {
  name: "Data type",
  icon: Icon.Binary,
  decode: (node) => {
    if (View.Filter.soleField(node) !== "dataType" || node.dataType == null)
      return null;
    const { eq, in: types } = node.dataType;
    // A stored filter can hold a data type as its plain value, so each name goes
    // through the constructor.
    const crude = eq != null ? [eq] : (types ?? []);
    return crude.length === 0 ? null : crude.map((t) => new DataType(t).toString());
  },
  encode: (types) => ({ dataType: { in: types.map((t) => new DataType(t)) } }),
  Picker: DataTypePicker,
  Operator: AnyOf,
  Value: ({ value }) => <>{summarize(value.map(nameOf), "data types")}</>,
};

type Kind = "virtual" | "stored";

const KINDS: Record<Kind, string> = { virtual: "Virtual", stored: "Stored" };

const KindPicker = ({
  value,
  onChange,
}: View.Filter.PickerProps<Kind>): ReactElement => (
  <Select.Frame<Kind, undefined>
    value={value ?? undefined}
    onChange={(next: Kind) => onChange(next)}
    allowNone
  >
    <Select.Dialog surface="menu" aria-label="Kinds">
      <Select.List>
        <Select.Item<Kind> itemKey="virtual">
          <Icon.Virtual />
          {KINDS.virtual}
        </Select.Item>
        <Select.Item<Kind> itemKey="stored">
          <Icon.Database />
          {KINDS.stored}
        </Select.Item>
      </Select.List>
    </Select.Dialog>
  </Select.Frame>
);

const KIND: View.Filter.Property<channel.Where, Kind> = {
  name: "Kind",
  icon: Icon.Channel,
  decode: (node) => {
    if (View.Filter.soleField(node) !== "virtual" || node.virtual == null) return null;
    return node.virtual ? "virtual" : "stored";
  },
  encode: (kind) => ({ virtual: kind === "virtual" }),
  Picker: KindPicker,
  Operator: Is,
  Value: ({ value }) => <>{KINDS[value]}</>,
};

/** The properties of a channel view's filter. */
export const PROPERTIES: View.Filter.Properties<channel.Where> = {
  folder: FOLDER,
  dataType: DATA_TYPE,
  kind: KIND,
};
