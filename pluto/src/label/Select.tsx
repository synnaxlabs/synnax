// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Tag } from "@synnaxlabs/lyra/tag";
import { state } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Color } from "@/color";
import { Flux } from "@/flux";
import { EditButton } from "@/label/Edit";
import { useResultMultiple } from "@/label/queries";
import { HAUL_TYPE } from "@/label/types";

/**
 * An option of a label select: the color of the label, then its name. While hovered, it
 * ends with a button that edits the label in place of the options.
 */
export const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<label.Key>): ReactElement | null => {
  const item = List.useItem<label.Key, label.Label>(itemKey);
  const resolve = Color.useResolve();
  const { hovered } = Select.useItemState(itemKey);
  if (item == null) return null;
  return (
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon.Circle color={resolve(item.color)} size="2.5em" />
      {item.name}
      {hovered && <EditButton itemKey={itemKey} name={item.name} />}
    </Select.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectMultipleProps
  extends
    Omit<
      Select.MultipleProps<label.Key, label.Label | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Omit<Flux.UseListParams<label.RetrieveMultipleParams, label.Label>, "name"> {}

const MultipleTag = (
  props: Select.MultipleTagProps<label.Key>,
): ReactElement | null => {
  const { itemKey, onClick } = props;
  const item = List.useItem<label.Key, label.Label>(itemKey);
  const { onSelect } = Select.useItemState<label.Key>(itemKey);
  const resolve = Color.useResolve();
  if (item == null) return null;
  return (
    <Tag.Tag
      color={resolve(item.color)}
      onClose={onSelect}
      onClick={onClick}
      size="small"
    >
      {item.name}
    </Tag.Tag>
  );
};

const labelRenderTag = Component.renderProp(MultipleTag);

const SELECT_MULTIPLE_TRIGGER_PROPS: Select.MultipleTriggerProps<
  label.Key,
  label.Label | undefined
> = { variant: "text" };

interface BaseProps extends Omit<
  Select.MultipleProps<label.Key, label.Label | undefined>,
  "resourceName" | "children"
> {}

const Base = (props: BaseProps): ReactElement => (
  <Select.Multiple<label.Key, label.Label | undefined>
    haulType={HAUL_TYPE}
    renderTag={labelRenderTag}
    icon={<Icon.Label />}
    triggerProps={SELECT_MULTIPLE_TRIGGER_PROPS}
    variant="floating"
    {...props}
    resourceName="label"
  >
    {listItemRenderProp}
  </Select.Multiple>
);

/**
 * Selects labels. Until the user first opens it, it reads only the labels of its value:
 * the list of every label mounts on the first open, since a page can hold one of these
 * per item.
 */
export const SelectMultiple = ({
  filter,
  query,
  onVisibleChange,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const [opened, setOpened] = useState(
    rest.visible === true || rest.initialVisible === true,
  );
  if (rest.visible === true && !opened) setOpened(true);
  const list = Flux.useList((client: Synnax) => client.labels, {
    name: "labels",
    query: opened ? query : null,
    filter,
  });
  const labels = useResultMultiple({ keys: rest.value }).data ?? [];
  const getValueItem = List.createGetItem<label.Key, label.Label | undefined>(
    (key) => labels.find((l) => l.key === key),
    (keys) => labels.filter((l) => keys.includes(l.key)),
  );
  // It is closed until the first open, so the dialog's last state is false.
  const handleVisibleChange = (next: state.SetArg<boolean>): void => {
    if (!opened && state.resolve(next, false)) setOpened(true);
    onVisibleChange?.(next);
  };
  // A change to virtual mounts the trigger again, so the list stays static once open.
  return (
    <Base
      getItem={opened ? list.getItem : getValueItem}
      onFetchMore={opened ? list.more : undefined}
      onSearch={opened ? list.search : undefined}
      status={opened ? list.status : undefined}
      virtual={false}
      {...rest}
      data={opened ? list.data : rest.value}
      subscribe={opened ? list.subscribe : undefined}
      onVisibleChange={handleVisibleChange}
    />
  );
};

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<label.Key, label.Label | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Omit<Flux.UseListParams<label.RetrieveMultipleParams, label.Label>, "name"> {}

export const SelectSingle = ({
  filter,
  query,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.labels,
    { name: "labels", query, filter },
  );
  return (
    <Select.Single<label.Key, label.Label | undefined>
      getItem={getItem}
      onFetchMore={more}
      onSearch={search}
      status={status}
      haulType={HAUL_TYPE}
      {...rest}
      data={data}
      resourceName="label"
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
