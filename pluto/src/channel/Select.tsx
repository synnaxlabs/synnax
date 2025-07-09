// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { array, DataType, unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { Align } from "@/align";
import { useAliases } from "@/channel/AliasContext";
import { useList } from "@/channel/queries";
import { HAUL_TYPE } from "@/channel/types";
import { Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Text } from "@/text";

export const resolveIcon = (ch?: channel.Payload): Icon.FC => {
  if (ch == null) return Icon.Channel;
  if (channel.isCalculated(ch)) return Icon.Calculation;
  if (ch.isIndex) return Icon.Index;
  const dt = new DataType(ch.dataType);
  if (dt.isInteger) return Icon.Binary;
  if (dt.isFloat) return Icon.Decimal;
  if (dt.equals(DataType.STRING)) return Icon.String;
  if (dt.equals(DataType.JSON)) return Icon.JSON;
  return Icon.Channel;
};

const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<channel.Key>): ReactElement | null => {
    const item = List.useItem<channel.Key, channel.Channel>(itemKey);
    const [selected, onSelect] = Select.useItemState<channel.Key>(itemKey);
    const aliases = useAliases();
    const Icon = resolveIcon(item?.payload);
    const displayName = aliases[item?.key ?? 0] ?? item?.name ?? "";
    return (
      <List.Item itemKey={itemKey} onSelect={onSelect} selected={selected} {...rest}>
        <Align.Space direction="x" size="small" align="center">
          <Icon />
          <Text.Text level="p">{displayName}</Text.Text>
        </Align.Space>
      </List.Item>
    );
  },
);

const canDrop = (
  { items: entities }: DraggingState,
  value: channel.Key[] | readonly channel.Key[],
): boolean => {
  const f = Haul.filterByType(HAUL_TYPE, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as channel.Key));
};

const MultipleTag = ({
  itemKey,
  onDragStart,
}: Omit<Tag.TagProps, "onDragStart"> & {
  itemKey: channel.Key;
  onDragStart: (key: channel.Key) => void;
}): ReactElement => {
  const item = List.useItem<channel.Key, channel.Channel>(itemKey);
  const [, onSelect] = Select.useItemState(itemKey);
  const aliases = useAliases();
  const Icon = resolveIcon(item?.payload);
  const displayName = aliases[item?.key ?? 0] ?? item?.name ?? "";
  return (
    <Tag.Tag
      icon={<Icon />}
      onSelect={onSelect}
      onDragStart={() => onDragStart(itemKey)}
      draggable
    >
      {displayName}
    </Tag.Tag>
  );
};

export interface MultipleTriggerProps extends Align.SpaceExtensionProps {
  onTagDragStart: (key: channel.Key) => void;
}

const MultipleTrigger = ({ onTagDragStart }: MultipleTriggerProps): ReactElement => {
  const value = Select.useSelection<channel.Key>();
  return (
    <Align.Space x bordered>
      {value.map((v) => (
        <MultipleTag key={v} itemKey={v} onDragStart={onTagDragStart} />
      ))}
    </Align.Space>
  );
};

export interface SelectMultipleProps extends Select.MultipleProps<channel.Key> {
  searchOptions?: channel.RetrieveOptions;
}

export const SelectMultiple = ({
  onChange,
  className,
  value,
  searchOptions,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  const { onSelect, ...selectProps } = Select.useMultiple({ value, onChange, data });

  const {
    startDrag,
    onDragEnd: endDrag,
    ...dropProps
  } = Haul.useDragAndDrop({
    type: "Channel.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: Haul.useFilterByTypeCallback(
      HAUL_TYPE,
      ({ items }) => {
        onChange(
          unique.unique([
            ...array.toArray(value),
            ...(items.map((c) => c.key) as channel.Key[]),
          ]),
          { clicked: null, clickedIndex: null },
        );
        return items;
      },
      [onChange, value],
    ),
  });

  const handleSuccessfulDrop = useCallback(
    ({ dropped }: Haul.OnSuccessfulDropProps) => {
      onChange(
        array.toArray(value).filter((key) => !dropped.some((h) => h.key === key)),
        { clicked: null, clickedIndex: null },
      );
    },
    [onChange, value],
  );

  const onTagDragStart = useCallback(
    (key: channel.Key) => startDrag([{ key, type: HAUL_TYPE }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop],
  );

  const dragging = Haul.useDraggingState();
  return (
    <Select.Dialog<channel.Key, channel.Channel | undefined>
      className={CSS(
        className,
        CSS.dropRegion(canDrop(dragging, array.toArray(value))),
      )}
      value={value}
      onSelect={onSelect}
      useItem={useListItem}
      data={data}
      {...dropProps}
      {...selectProps}
      {...rest}
    >
      <MultipleTrigger onTagDragStart={onTagDragStart} />
      <DialogContent retrieve={retrieve} searchOptions={searchOptions} />
    </Select.Dialog>
  );
};

const SingleTrigger = (): ReactElement => {
  const [value] = Select.useSelection<channel.Key>();
  const item = List.useItem<channel.Key, channel.Channel>(value);
  const Icon = resolveIcon(item?.payload);
  return (
    <Dialog.Trigger>
      <Align.Space direction="x" size="small" align="center">
        <Icon />
        <Text.Text level="p">{item?.name}</Text.Text>
      </Align.Space>
    </Dialog.Trigger>
  );
};

const DialogContent = ({
  retrieve,
  searchOptions,
}: Pick<ReturnType<typeof useList>, "retrieve"> & {
  searchOptions?: channel.RetrieveOptions;
}): ReactElement => {
  const [search, setSearch] = useState("");
  return (
    <Dialog.Frame>
      <Input.Text
        value={search}
        onChange={(term) => {
          setSearch(term);
          retrieve((p) => ({ ...p, term, ...searchOptions }));
        }}
        placeholder="Search channels..."
      />
      <List.Items>{listItemRenderProp}</List.Items>
    </Dialog.Frame>
  );
};

export interface SelectSingleProps extends Select.SingleProps<channel.Key> {
  searchOptions?: channel.RetrieveOptions;
}

export const SelectSingle = ({
  onChange,
  value,
  className,
  searchOptions,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  const { onSelect, ...selectProps } = Select.useSingle({ value, onChange, data });

  const { startDrag, ...dragProps } = Haul.useDragAndDrop({
    type: "Channel.SelectSingle",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: Haul.useFilterByTypeCallback(
      HAUL_TYPE,
      ({ items }) => {
        if (items.length !== 0) onSelect(items[0].key as channel.Key);
        return items;
      },
      [onSelect],
    ),
  });

  const dragging = Haul.useDraggingState();
  const onDragStart = useCallback(
    () => value != null && startDrag([{ type: HAUL_TYPE, key: value }]),
    [startDrag, value],
  );

  return (
    <Select.Dialog<channel.Key, channel.Channel | undefined>
      className={CSS(
        className,
        CSS.dropRegion(canDrop(dragging, array.toArray(value))),
      )}
      value={value}
      onSelect={onSelect}
      useItem={useListItem}
      data={data}
      onDragStart={onDragStart}
      {...dragProps}
      {...selectProps}
      {...rest}
    >
      <SingleTrigger />
      <DialogContent retrieve={retrieve} searchOptions={searchOptions} />
    </Select.Dialog>
  );
};
