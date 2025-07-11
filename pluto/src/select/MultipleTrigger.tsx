// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { array, type record, unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Button } from "@/button";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Text } from "@/text";

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
    <Tag.Tag
      onClose={onSelect}
      onDragStart={() => onDragStart(itemKey)}
      draggable
      size="small"
      icon={<Icon.Range />}
    >
      {item?.name}
    </Tag.Tag>
  );
};

export interface MultipleTriggerProps {
  haulType?: string;
  disabled?: boolean;
}

export const canDrop = <K extends record.Key>(
  { items: entities }: Haul.DraggingState,
  haulType: string,
  value: K[] | readonly K[],
  disabled?: boolean,
): boolean => {
  if (haulType === "" || disabled === true) return false;
  const f = Haul.filterByType(haulType, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as K));
};

export const MultipleTrigger = ({
  haulType = "",
  disabled,
}: MultipleTriggerProps): ReactElement => {
  const value = Select.useSelection<ranger.Key>();
  const { open } = Dialog.useContext();
  const { onSelect } = Select.useContext();
  const { startDrag, ...dropProps } = Haul.useDragAndDrop({
    type: haulType,
    canDrop: useCallback(
      (hauled) => canDrop(hauled, haulType, array.toArray(value), disabled),
      [haulType, value, disabled],
    ),
    onDrop: Haul.useFilterByTypeCallback(
      haulType,
      ({ items }) => {
        onSelect(
          ...unique.unique([
            ...array.toArray(value),
            ...(items.map((c) => c.key) as ranger.Keys),
          ]),
        );
        return items;
      },
      [onSelect, value],
    ),
  });

  const handleSuccessfulDrop = useCallback(
    ({ dropped }: Haul.OnSuccessfulDropProps) => {
      onSelect(
        ...array.toArray(value).filter((key) => !dropped.some((h) => h.key === key)),
      );
    },
    [onSelect, value],
  );

  const onTagDragStart = useCallback(
    (key: ranger.Key) => startDrag([{ key, type: haulType }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop, haulType],
  );
  return (
    <Tag.Tags
      onClick={open}
      actions={
        <Button.Icon size="small" onClick={() => open()} shade={2} variant="outlined">
          <Icon.Add />
        </Button.Icon>
      }
      {...dropProps}
    >
      {value.length === 0 && (
        <Text.Text level="p" shade={8} weight={400} style={{ marginLeft: "1rem" }}>
          Select Ranges...
        </Text.Text>
      )}
      {value.map((v) => (
        <MultipleTag key={v} itemKey={v} onDragStart={onTagDragStart} />
      ))}
    </Tag.Tags>
  );
};
