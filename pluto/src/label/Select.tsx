// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { array, unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { Align } from "@/align";
import { Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { useList } from "@/label/queries";
import { HAUL_TYPE } from "@/label/types";
import { List } from "@/list";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Text } from "@/text";

const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<label.Key>): ReactElement | null => {
    const item = List.useItem<label.Key, label.Label>(itemKey);
    const [selected, onSelect] = Select.useItemState<label.Key>(itemKey);
    return (
      <List.Item itemKey={itemKey} onSelect={onSelect} selected={selected} {...rest}>
        <Icon.Circle color={item?.color} size="1.5rem" />
        <Text.Text level="p">{item?.name}</Text.Text>
      </List.Item>
    );
  },
);

const canDrop = (
  { items: entities }: DraggingState,
  value: label.Key[] | readonly label.Key[],
): boolean => {
  const f = Haul.filterByType(HAUL_TYPE, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as label.Key));
};

const MultipleTag = ({
  itemKey,
  onDragStart,
}: Omit<Tag.TagProps, "onDragStart"> & {
  itemKey: label.Key;
  onDragStart: (key: label.Key) => void;
}): ReactElement => {
  const item = List.useItem<label.Key, label.Label>(itemKey);
  const [, onSelect] = Select.useItemState(itemKey);
  return (
    <Tag.Tag
      color={item?.color}
      onSelect={onSelect}
      onDragStart={() => onDragStart(itemKey)}
      draggable
    >
      {item?.name}
    </Tag.Tag>
  );
};

export interface MultipleTriggerProps extends Align.SpaceExtensionProps {
  onTagDragStart: (key: label.Key) => void;
}

const MultipleTrigger = ({ onTagDragStart }: MultipleTriggerProps): ReactElement => {
  const value = Select.useSelection<label.Key>();
  return (
    <Align.Space x bordered>
      {value.map((v) => (
        <MultipleTag key={v} itemKey={v} onDragStart={onTagDragStart} />
      ))}
    </Align.Space>
  );
};

export interface SelectMultipleProps extends Select.MultipleProps<label.Key> {}

export const SelectMultiple = ({
  onChange,
  className,
  value,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  const { onSelect, ...selectProps } = Select.useMultiple({ value, onChange, data });
  const {
    startDrag,
    onDragEnd: endDrag,
    ...dropProps
  } = Haul.useDragAndDrop({
    type: "Label.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: Haul.useFilterByTypeCallback(
      HAUL_TYPE,
      ({ items }) => {
        onChange(
          unique.unique([
            ...array.toArray(value),
            ...(items.map((c) => c.key) as label.Key[]),
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
    (key: label.Key) => startDrag([{ key, type: HAUL_TYPE }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop],
  );

  const dragging = Haul.useDraggingState();
  return (
    <Select.Dialog<label.Key, label.Label | undefined>
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
      <DialogContent retrieve={retrieve} />
    </Select.Dialog>
  );
};

const SingleTrigger = (): ReactElement => {
  const [value] = Select.useSelection<label.Key>();
  const item = List.useItem<label.Key, label.Label>(value);
  return (
    <Dialog.Trigger>
      <Align.Space direction="x" size="small" align="center">
        <Icon.Circle color={item?.color} size="1.5rem" />
        <Text.Text level="p">{item?.name}</Text.Text>
      </Align.Space>
    </Dialog.Trigger>
  );
};

const DialogContent = ({
  retrieve,
}: Pick<ReturnType<typeof useList>, "retrieve"> & {}): ReactElement => {
  const [search, setSearch] = useState("");
  return (
    <Dialog.Content>
      <Input.Text
        value={search}
        onChange={(v) => {
          setSearch(v);
          retrieve((prev) => ({ ...prev, term: v }));
        }}
        placeholder="Search labels..."
      />
      <List.Items>{listItemRenderProp}</List.Items>
    </Dialog.Content>
  );
};

export interface SelectSingleProps extends Select.SingleProps<label.Key> {}

export const SelectSingle = ({
  onChange,
  value,
  className,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  const { onSelect, ...selectProps } = Select.useSingle({ value, onChange, data });
  const { startDrag, ...dragProps } = Haul.useDragAndDrop({
    type: "Label.SelectSingle",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: Haul.useFilterByTypeCallback(
      HAUL_TYPE,
      ({ items }) => {
        if (items.length !== 0) onSelect(items[0].key as label.Key);
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
    <Select.Dialog<label.Key, label.Label | undefined>
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
      <DialogContent retrieve={retrieve} />
    </Select.Dialog>
  );
};
