// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type record, unique } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { Component } from "@/component";
import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Text } from "@/text";

export interface MultipleEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

export interface MultipleTagProps<K extends record.Key>
  extends Omit<Tag.TagProps, "onDragStart"> {
  itemKey: K;
  onDragStart: (key: K) => void;
}

const MultipleTag = <K extends record.Key, E extends MultipleEntry<K>>({
  itemKey,
  icon,
  onDragStart,
}: MultipleTagProps<K>): ReactElement | null => {
  const item = List.useItem<K, E>(itemKey);
  const { onSelect } = Select.useItemState(itemKey);
  return (
    <Tag.Tag
      onClose={onSelect}
      onDragStart={() => onDragStart(itemKey)}
      draggable
      size="small"
      status={item == null ? "error" : undefined}
      icon={item?.icon ?? icon}
    >
      {item?.name ?? itemKey}
    </Tag.Tag>
  );
};

const multipleTag = Component.renderProp(MultipleTag);

export interface MultipleTriggerProps<K extends record.Key>
  extends Pick<Button.ButtonProps, "variant" | "disabled"> {
  haulType?: string;
  placeholder?: ReactNode;
  icon?: Icon.ReactElement;
  children?: RenderProp<MultipleTagProps<K>>;
}

export const staticCanDrop = <K extends record.Key>(
  { items: entities }: Haul.DraggingState,
  haulType: string,
  value: K[] | readonly K[],
  disabled?: boolean,
): boolean => {
  if (haulType === "" || disabled === true) return false;
  const f = Haul.filterByType(haulType, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as K));
};

export const MultipleTrigger = <K extends record.Key>({
  haulType = "",
  disabled,
  placeholder = "Select...",
  variant = "outlined",
  icon,
  children = multipleTag as unknown as RenderProp<MultipleTagProps<K>>,
}: MultipleTriggerProps<K>): ReactElement => {
  const value = Select.useSelection<K>();
  const valueRef = useSyncedRef(value);
  const { setSelected } = Select.useContext<K>();
  const { toggle, visible } = Dialog.useContext();
  const canDrop = useCallback(
    (hauled: Haul.DraggingState) =>
      staticCanDrop(hauled, haulType, array.toArray(value), disabled),
    [haulType, value, disabled],
  );
  const { startDrag, ...dropProps } = Haul.useDragAndDrop({
    type: haulType,
    canDrop,
    onDrop: Haul.useFilterByTypeCallback(
      haulType,
      ({ items }) => {
        const v = array.toArray(valueRef.current);
        setSelected(
          unique.unique([...array.toArray(v), ...(items.map((c) => c.key) as K[])]),
        );
        return items;
      },
      [setSelected],
    ),
  });

  const handleSuccessfulDrop = useCallback(
    ({ dropped }: Haul.OnSuccessfulDropProps) => {
      const res = value.filter((key) => !dropped.some((h) => h.key === key));
      setSelected(res);
    },
    [setSelected, value],
  );

  const onTagDragStart = useCallback(
    (key: K) => {
      startDrag([{ key, type: haulType }], handleSuccessfulDrop);
    },
    [startDrag, handleSuccessfulDrop, haulType],
  );
  const dragging = Haul.useDraggingState();
  const showAddButton = variant === "text" && value.length !== 0;

  return (
    <Tag.Tags
      full="x"
      onClick={() => {
        if (!showAddButton) toggle();
      }}
      {...dropProps}
      className={CSS(
        CSS.dropRegion(canDrop(dragging)),
        CSS.BE("dialog", "trigger"),
        CSS.BM("variant", variant),
      )}
      variant={variant}
      preventClick={showAddButton}
      grow
    >
      {value.length === 0 && (
        <Text.Text color={8} weight={400} style={{ padding: "0 1rem" }}>
          {icon}
          {placeholder}
        </Text.Text>
      )}
      {value.map((v) =>
        children({ key: v, itemKey: v, onDragStart: onTagDragStart, icon }),
      )}
      {variant !== "text" && (
        <Caret.Animated
          className={CSS.level("p")}
          enabled={visible}
          enabledLoc="bottom"
          disabledLoc="left"
          color={8}
        />
      )}
      {showAddButton && (
        <Button.Button
          variant={variant}
          onClick={(e) => {
            e.stopPropagation();
            toggle();
          }}
        >
          <Icon.Add color={8} />
        </Button.Button>
      )}
    </Tag.Tags>
  );
};
