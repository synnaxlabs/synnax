// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { array, type AsyncTermSearcher, unique } from "@synnaxlabs/x";
import { type DragEvent, type ReactElement, useCallback, useId, useMemo } from "react";

import { CSS } from "@/css";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { type List } from "@/list";
import { useList } from "@/ranger/queries";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { componentRenderProp } from "@/util/renderProp";

const canDrop = (
  { items: entities }: DraggingState,
  value: ranger.Key[] | readonly ranger.Key[],
): boolean => {
  const f = Haul.filterByType(HAUL_TYPE, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as ranger.Key));
};

export interface SelectMultipleProps
  extends Omit<
    Select.MultipleProps<ranger.Key, ranger.Payload>,
    "columns" | "searcher"
  > {}

export const SelectMultiple = ({
  onChange,
  className,
  value,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const {
    startDrag,
    onDragEnd: endDrag,
    ...dropProps
  } = Haul.useDragAndDrop({
    type: "Ranger.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        if (dropped.length === 0) return [];
        const v = unique.unique([
          ...array.toArray(value),
          ...(dropped.map((c) => c.key) as ranger.Keys),
        ]);
        onChange(v, {
          clicked: null,
          entries: [],
        });
        return dropped;
      },
      [onChange, value],
    ),
  });
  const dragging = Haul.useDraggingState();

  const handleSuccessfulDrop = useCallback(
    ({ dropped }: Haul.OnSuccessfulDropProps) => {
      onChange(
        array.toArray(value).filter((key) => !dropped.some((h) => h.key === key)),
        { clicked: null, entries: [] },
      );
    },
    [onChange, value],
  );

  const onDragStart = useCallback(
    (_: DragEvent<HTMLDivElement>, key: ranger.Key) =>
      startDrag([{ key, type: HAUL_TYPE }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop],
  );

  return (
    <Select.Multiple
      className={CSS(
        className,
        CSS.dropRegion(canDrop(dragging, array.toArray(value))),
      )}
      value={value}
      onTagDragStart={onDragStart}
      onTagDragEnd={endDrag}
      searcher={client?.ranges}
      onChange={onChange}
      columns={rangeCols}
      emptyContent={emptyContent}
      entryRenderKey="name"
      {...dropProps}
      {...rest}
    />
  );
};

export interface SelectSingleProps
  extends Omit<
    Select.SingleProps<ranger.Key, ranger.Payload>,
    "columns" | "children"
  > {}

const SingleTrigger = ({
  value,
  useItem,
  onClick,
}: Select.TriggerProps<ranger.Key, ranger.Payload | undefined>): ReactElement => <></>;

const ListItem = ({
  key,
  index,
  itemKey,
  translate,
  useItem,
}: List.ItemProps<ranger.Key, ranger.Payload | undefined>): ReactElement => <></>;

const listItemRenderProp = componentRenderProp(ListItem);

const singleTriggerRenderProp = componentRenderProp(SingleTrigger);

export const SelectSingle = ({
  onChange,
  value,
  className,
  data: _,
  useItem: __,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, useListItem } = useList();
  const id = useId();
  const sourceAndTarget: Haul.Item = useMemo(
    () => ({ key: id, type: "Ranger.SelectMultiple" }),
    [id],
  );
  const { startDrag, ...dragProps } = Haul.useDragAndDrop({
    type: "Ranger.SelectSingle",
    canDrop: useCallback((hauled) => canDrop(hauled, array.toArray(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const ch = Haul.filterByType(HAUL_TYPE, items);
        if (ch.length === 0) return [];
        onChange(ch[0].key as ranger.Key, { clicked: null, clickedIndex: 0 });
        return ch;
      },
      [sourceAndTarget, onChange],
    ),
  });
  const dragging = Haul.useDraggingState();
  const onDragStart = useCallback(
    () => value != null && startDrag([{ type: HAUL_TYPE, key: value }]),
    [startDrag, value],
  );
  return (
    <Select.Single<ranger.Key, ranger.Payload | undefined>
      className={CSS(
        className,
        CSS.dropRegion(canDrop(dragging, array.toArray(value))),
      )}
      value={value}
      onChange={onChange}
      useItem={useListItem}
      data={data}
      onDragStart={onDragStart}
      {...dragProps}
      {...rest}
    >
      {singleTriggerRenderProp}
      {listItemRenderProp}
    </Select.Single>
  );
};
