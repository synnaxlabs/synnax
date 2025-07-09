// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { array, unique } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback, useState } from "react";

import { Align } from "@/align";
import { Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { useList } from "@/ranger/queries";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Text } from "@/text";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<ranger.Key>): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  const { selected, onSelect, hovered } = Select.useItemState<ranger.Key>(itemKey);
  return (
    <List.Item
      itemKey={itemKey}
      onSelect={onSelect}
      selected={selected}
      hovered={hovered}
      {...rest}
    >
      <Text.Text level="p">{item?.name}</Text.Text>
    </List.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

const canDrop = (
  { items: entities }: DraggingState,
  value: ranger.Key[] | readonly ranger.Key[],
): boolean => {
  const f = Haul.filterByType(HAUL_TYPE, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as ranger.Key));
};

const MultipleTag = ({
  itemKey,
  onDragStart,
}: Omit<Tag.TagProps, "onDragStart"> & {
  itemKey: ranger.Key;
  onDragStart: (key: ranger.Key) => void;
}): ReactElement => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  const { onSelect } = Select.useItemState(itemKey);
  return (
    <Tag.Tag onSelect={onSelect} onDragStart={() => onDragStart(itemKey)} draggable>
      {item?.name}
    </Tag.Tag>
  );
};

export interface MultipleTriggerProps extends Align.SpaceExtensionProps {
  onTagDragStart: (key: ranger.Key) => void;
}

const MultipleTrigger = ({ onTagDragStart }: MultipleTriggerProps): ReactElement => {
  const value = Select.useSelection<ranger.Key>();
  return (
    <Align.Space x bordered>
      {value.map((v) => (
        <MultipleTag key={v} itemKey={v} onDragStart={onTagDragStart} />
      ))}
    </Align.Space>
  );
};

export interface SelectMultipleProps extends Select.MultipleProps<ranger.Key> {}

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
    type: "Ranger.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: Haul.useFilterByTypeCallback(
      HAUL_TYPE,
      ({ items }) => {
        onChange(
          unique.unique([
            ...array.toArray(value),
            ...(items.map((c) => c.key) as ranger.Keys),
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
    (key: ranger.Key) => startDrag([{ key, type: HAUL_TYPE }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop],
  );

  const dragging = Haul.useDraggingState();
  return (
    <Select.Dialog<ranger.Key, ranger.Payload | undefined>
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

export interface SingleTriggerProps extends Align.SpaceExtensionProps {
  placeholder?: ReactNode;
  triggerIcon?: Icon.ReactElement;
}

const SingleTrigger = ({
  placeholder = "Select Range",
  triggerIcon = <Icon.Range />,
}: SingleTriggerProps): ReactElement => {
  const [value] = Select.useSelection<ranger.Key>();
  const item = List.useItem<ranger.Key, ranger.Payload>(value);
  return (
    <Dialog.Trigger startIcon={triggerIcon} iconSpacing="small">
      {item?.name ?? placeholder}
    </Dialog.Trigger>
  );
};

const DialogContent = ({
  retrieve,
}: Pick<ReturnType<typeof useList>, "retrieve">): ReactElement => {
  const [search, setSearch] = useState("");
  return (
    <Dialog.Dialog style={{ width: 500 }}>
      <Input.Text
        value={search}
        borderShade={5}
        autoFocus
        placeholder="Search Ranges..."
        onChange={(v) => {
          setSearch(v);
          retrieve((prev) => ({ ...prev, term: v }));
        }}
      />
      <List.Items bordered borderShade={5} style={{ minHeight: 300 }}>
        {listItemRenderProp}
      </List.Items>
    </Dialog.Dialog>
  );
};

export interface SelectSingleProps
  extends Select.SingleProps<ranger.Key>,
    SingleTriggerProps {
  filter?: (item: ranger.Payload) => boolean;
}

export const SelectSingle = ({
  onChange,
  value,
  className,
  filter,
  triggerIcon,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList({ filter });
  const { onSelect, ...selectProps } = Select.useSingle({ value, onChange, data });
  const { startDrag, ...dragProps } = Haul.useDragAndDrop({
    type: "Ranger.SelectSingle",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: Haul.useFilterByTypeCallback(
      HAUL_TYPE,
      ({ items }) => {
        if (items.length !== 0) onSelect(items[0].key as ranger.Key);
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
    <Select.Dialog<ranger.Key, ranger.Payload | undefined>
      className={CSS(
        className,
        CSS.dropRegion(canDrop(dragging, array.toArray(value))),
      )}
      variant="floating"
      value={value}
      onSelect={onSelect}
      useItem={useListItem}
      data={data}
      onDragStart={onDragStart}
      {...dragProps}
      {...selectProps}
      {...rest}
    >
      <SingleTrigger triggerIcon={triggerIcon} />
      <DialogContent retrieve={retrieve} />
    </Select.Dialog>
  );
};
