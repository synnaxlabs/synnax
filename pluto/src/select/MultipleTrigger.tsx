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
}

export const canDrop = <K extends record.Key>(
  { items: entities }: Haul.DraggingState,
  haulType: string,
  value: K[] | readonly K[],
): boolean => {
  if (haulType === "") return false;
  const f = Haul.filterByType(haulType, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as K));
};

export const MultipleTrigger = ({
  haulType = "",
}: MultipleTriggerProps): ReactElement => {
  const value = Select.useSelection<ranger.Key>();
  const { open } = Dialog.useContext();
  const { onSelect } = Select.useContext();
  const { startDrag, ...dropProps } = Haul.useDragAndDrop({
    type: haulType,
    canDrop: useCallback(
      (hauled) => canDrop(hauled, haulType, array.toArray(value)),
      [haulType, value],
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
