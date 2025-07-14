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
import { type ReactElement, type ReactNode, useCallback } from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { Component } from "@/component";
import { type RenderProp } from "@/component/renderProp";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { type Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Text } from "@/text";

interface MultipleTagProps extends Omit<Tag.TagProps, "onDragStart"> {
  itemKey: ranger.Key;
  onDragStart: (key: ranger.Key) => void;
}

const MultipleTag = ({
  itemKey,
  onDragStart,
  icon,
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
      icon={icon}
    >
      {item?.name}
    </Tag.Tag>
  );
};

const multipleTag = Component.renderProp(MultipleTag);

export interface MultipleTriggerProps {
  haulType?: string;
  disabled?: boolean;
  placeholder?: ReactNode;
  icon?: Icon.ReactElement;
  children?: RenderProp<MultipleTagProps>;
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
  placeholder = "Select...",
  icon,
  children = multipleTag,
}: MultipleTriggerProps): ReactElement => {
  const value = Select.useSelection<ranger.Key>();
  const { toggle, visible } = Dialog.useContext();
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
      onClick={toggle}
      {...dropProps}
      actions={
        <Button.Icon variant="outlined" onClick={toggle}>
          <Caret.Animated
            enabled={visible}
            enabledLoc="bottom"
            disabledLoc="left"
            color={8}
          />
        </Button.Icon>
      }
      grow
    >
      {value.length === 0 && (
        <Text.Text level="p" shade={8} weight={400} style={{ marginLeft: "1rem" }}>
          {placeholder}
        </Text.Text>
      )}
      {value.map((v) =>
        children({ key: v, itemKey: v, onDragStart: onTagDragStart, icon }),
      )}
    </Tag.Tags>
  );
};
