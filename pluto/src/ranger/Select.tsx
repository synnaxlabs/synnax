// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type AsyncTermSearcher, nullToArr, toArray, unique } from "@synnaxlabs/x";
import { type DragEvent, type ReactElement, useCallback, useId, useMemo } from "react";

import { CSS } from "@/css";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { type List } from "@/list";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

const rangeCols: Array<List.ColumnSpec<ranger.Key, ranger.Payload>> = [
  { key: "name", name: "Name" },
];

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
  ...props
}: SelectMultipleProps): ReactElement => {
  const client = Synnax.use();
  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4" style={{ height: 150 }}>
        No client available
      </Status.Text.Centered>
    );

  const {
    startDrag,
    onDragEnd: endDrag,
    ...dropProps
  } = Haul.useDragAndDrop({
    type: "Ranger.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, toArray(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        if (dropped.length === 0) return [];
        const v = unique.unique([
          ...toArray(value),
          ...(dropped.map((c) => c.key) as ranger.Keys),
        ]);
        onChange(v, {
          clickedIndex: null,
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
        toArray(value).filter((key) => !dropped.some((h) => h.key === key)),
        {
          clickedIndex: null,
          clicked: null,
          entries: [],
        },
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
      className={CSS(className, CSS.dropRegion(canDrop(dragging, toArray(value))))}
      value={value}
      onTagDragStart={onDragStart}
      onTagDragEnd={endDrag}
      searcher={client?.ranges}
      onChange={onChange}
      columns={rangeCols}
      emptyContent={emptyContent}
      entryRenderKey={"name"}
      {...dropProps}
      {...props}
    />
  );
};

export interface SelectSingleProps
  extends Omit<Select.SingleProps<ranger.Key, ranger.Payload>, "columns"> {}

interface UseSingleReturn extends Omit<Haul.UseDragReturn, "startDrag"> {
  emptyContent?: ReactElement;
  dragging: DraggingState;
  onDragStart: (e: DragEvent<HTMLDivElement>) => void;
  searcher?: AsyncTermSearcher<string, ranger.Key, ranger.Payload>;
}

const useSingle = ({
  value,
  onChange,
}: Pick<SelectSingleProps, "onChange" | "value">): UseSingleReturn => {
  const client = Synnax.use();
  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4" style={{ height: 150 }}>
        No client available
      </Status.Text.Centered>
    );

  const id = useId();
  const sourceAndTarget: Haul.Item = useMemo(
    () => ({ key: id, type: "Ranger.SelectMultiple" }),
    [id],
  );

  const { startDrag, ...dragProps } = Haul.useDragAndDrop({
    type: "Ranger.SelectSingle",
    canDrop: useCallback((hauled) => canDrop(hauled, nullToArr(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const ch = Haul.filterByType(HAUL_TYPE, items);
        if (ch.length === 0) return [];
        onChange(ch[0].key as ranger.Key, {
          clickedIndex: null,
          clicked: null,
          entries: [],
        });
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
  return {
    emptyContent,
    dragging,
    ...dragProps,
    onDragStart,
    searcher: client?.ranges,
  };
};

export const SelectSingle = ({
  onChange,
  value,
  className,
  data,
  ...props
}: SelectSingleProps): ReactElement => {
  const { dragging, ...dragProps } = useSingle({ value, onChange });
  return (
    <Select.Single<ranger.Key, ranger.Payload>
      data={data}
      className={CSS(className, CSS.dropRegion(canDrop(dragging, nullToArr(value))))}
      value={value}
      onChange={onChange}
      columns={rangeCols}
      entryRenderKey={"name"}
      {...dragProps}
      {...props}
    />
  );
};

export const SelectButton = ({
  data,
  value,
  onChange,
  ...props
}: SelectSingleProps): ReactElement => {
  const { dragging, ...dragProps } = useSingle({ value, onChange });
  return (
    <Select.Single<ranger.Key, ranger.Payload>
      data={data}
      value={value as string}
      onChange={onChange}
      columns={rangeCols}
      entryRenderKey={"name"}
      {...dragProps}
      {...props}
    />
  );
};
