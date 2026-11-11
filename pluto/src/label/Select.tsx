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
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Tag } from "@synnaxlabs/lyra/tag";
import { state } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Flux } from "@/flux";
import { useResultMultiple } from "@/label/queries";
import { HAUL_TYPE } from "@/label/types";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<label.Key>): ReactElement | null => {
  const item = List.useItem<label.Key, label.Label>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon.Circle color={item?.color} size="2.5em" />
      {item?.name}
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
  const { itemKey } = props;
  const item = List.useItem<label.Key, label.Label>(itemKey);
  const { onSelect } = Select.useItemState<label.Key>(itemKey);
  if (item == null) return null;
  return (
    <Tag.Tag color={item.color} onClose={onSelect} size="small">
      {item.name}
    </Tag.Tag>
  );
};

const labelRenderTag = Component.renderProp(MultipleTag);

const SELECT_MULTIPLE_TRIGGER_PROPS: Select.MultipleTriggerProps<label.Key> = {
  variant: "text",
};

export const SelectMultiple = ({
  filter,
  query,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.labels,
    { name: "labels", query, filter },
  );
  return (
    <Select.Multiple<label.Key, label.Label | undefined>
      haulType={HAUL_TYPE}
      getItem={getItem}
      onFetchMore={more}
      onSearch={search}
      status={status}
      renderTag={labelRenderTag}
      icon={<Icon.Label />}
      triggerProps={SELECT_MULTIPLE_TRIGGER_PROPS}
      variant="floating"
      {...rest}
      data={data}
      subscribe={subscribe}
      resourceName="label"
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectMultipleTriggerProps extends Pick<
  SelectMultipleProps,
  | "value"
  | "onChange"
  | "disabled"
  | "preview"
  | "triggerProps"
  | "className"
  | "style"
  | "variant"
  | "location"
> {
  /** Called when the user asks to open the selection. */
  onOpen: () => void;
}

/**
 * The trigger of a {@link SelectMultiple}, alone: it shows the labels of the value and
 * removes one on its tag, with no list of labels and no dialog. Mount the
 * {@link SelectMultiple} when it calls onOpen, which costs far more. It takes only the
 * props that shape a closed select, so it accepts all the props of a
 * {@link SelectMultiple}.
 */
export const SelectMultipleTrigger = ({
  value,
  onChange,
  disabled,
  preview,
  triggerProps,
  className,
  style,
  variant = "floating",
  location,
  onOpen,
}: SelectMultipleTriggerProps): ReactElement => {
  const labels = useResultMultiple({ keys: value }).data ?? [];
  const placeholder = Select.usePlaceholder("labels");
  const getItem = List.createGetItem<label.Key, label.Label | undefined>(
    (key) => labels.find((l) => l.key === key),
    (keys) => labels.filter((l) => keys.includes(l.key)),
  );
  const handleVisibleChange = (next: state.SetArg<boolean>): void => {
    if (state.executeSetter(next, false)) onOpen();
  };
  return (
    <Dialog.Frame
      className={className}
      style={style}
      variant={variant}
      location={location}
      visible={false}
      onVisibleChange={handleVisibleChange}
    >
      <Select.Frame<label.Key, label.Label | undefined>
        multiple
        value={value}
        onChange={onChange}
        data={value}
        getItem={getItem}
        virtual={false}
      >
        <Select.MultipleTrigger<label.Key, label.Label | undefined>
          haulType={HAUL_TYPE}
          icon={<Icon.Label />}
          placeholder={placeholder}
          aria-label="Labels"
          disabled={disabled}
          preview={preview}
          {...SELECT_MULTIPLE_TRIGGER_PROPS}
          {...triggerProps}
        >
          {labelRenderTag}
        </Select.MultipleTrigger>
      </Select.Frame>
    </Dialog.Frame>
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
