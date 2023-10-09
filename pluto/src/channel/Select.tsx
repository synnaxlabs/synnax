// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DragEvent, type ReactElement, useCallback, useId, useMemo } from "react";

import { type channel } from "@synnaxlabs/client";
import { unique } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { type List } from "@/list";
import { Select } from "@/select";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

import { HAUL_TYPE } from "./types";

const channelColumns: Array<List.ColumnSpec<channel.Key, channel.Payload>> = [
  {
    key: "name",
    name: "Name",
  },
  {
    key: "rate",
    name: "Rate",
  },
  {
    key: "dataType",
    name: "Data Type",
  },
  {
    key: "index",
    name: "Index",
  },
  {
    key: "key",
    name: "Key",
  },
  {
    key: "isIndex",
    name: "Is Index",
  },
];

const canDrop = (
  { items: entities }: DraggingState,
  value: channel.Key[] | readonly channel.Key[],
): boolean => {
  const f = Haul.filterByType(HAUL_TYPE, entities);
  return f.length > 0 && !f.every((h) => value.includes(h.key as channel.Key));
};

export interface SelectMultipleProps
  extends Omit<
    Select.MultipleProps<channel.Key, channel.Payload>,
    "columns" | "searcher"
  > {
  columns?: string[];
}

const DEFAULT_FILTER = ["name"];

export const SelectMultiple = ({
  columns: filter = DEFAULT_FILTER,
  onChange,
  value,
  className,
  ...props
}: SelectMultipleProps): ReactElement => {
  const client = Synnax.use();
  const columns = useMemo(() => {
    if (filter.length === 0) return channelColumns;
    return channelColumns.filter((column) => filter.includes(column.key));
  }, [filter]);

  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4">
        No client available
      </Status.Text.Centered>
    );

  const {
    startDrag,
    onDragEnd: endDrag,
    ...dropProps
  } = Haul.useDragAndDrop({
    type: "Channel.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, value), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        if (dropped.length === 0) return [];
        onChange(unique([...value, ...(dropped.map((c) => c.key) as channel.Keys)]));
        return dropped;
      },
      [onChange, value],
    ),
  });
  const dragging = Haul.useDraggingState();

  const handleSuccessfulDrop = useCallback(
    ({ dropped }: Haul.OnSuccessfulDropProps) => {
      onChange(value.filter((key) => !dropped.some((h) => h.key === key)));
    },
    [onChange, value],
  );

  const onDragStart = useCallback(
    (_: DragEvent<HTMLDivElement>, key: channel.Key) =>
      startDrag([{ key, type: HAUL_TYPE }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop],
  );

  return (
    <Select.Multiple
      className={CSS(className, CSS.dropRegion(canDrop(dragging, value)))}
      value={value}
      onTagDragStart={onDragStart}
      onTagDragEnd={endDrag}
      searcher={client?.channels}
      onChange={onChange}
      columns={columns}
      emptyContent={emptyContent}
      tagKey={"name"}
      {...dropProps}
      {...props}
    />
  );
};

export interface SelectSingleProps
  extends Omit<Select.SingleProps<channel.Key, channel.Payload>, "columns"> {
  columns?: string[];
}

export const SelectSingle = ({
  columns: filter = [],
  onChange,
  value,
  className,
  ...props
}: SelectSingleProps): ReactElement => {
  const client = Synnax.use();
  const columns = useMemo(() => {
    if (filter.length === 0) return channelColumns;
    return channelColumns.filter((column) => filter.includes(column.key));
  }, [filter]);

  const emptyContent =
    client != null ? undefined : (
      <Status.Text.Centered variant="error" level="h4" style={{ height: 150 }}>
        No client available
      </Status.Text.Centered>
    );

  const id = useId();
  const sourceAndTarget: Haul.Item = useMemo(
    () => ({ key: id, type: "Channel.SelectMultiple" }),
    [id],
  );

  const {
    startDrag,
    onDragEnd: endDrag,
    ...dragProps
  } = Haul.useDragAndDrop({
    type: "Channel.SelectSingle",
    canDrop: useCallback((hauled) => canDrop(hauled, [value]), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const ch = Haul.filterByType(HAUL_TYPE, items);
        if (ch.length === 0) return [];
        onChange(ch[0].key as channel.Key);
        return ch;
      },
      [sourceAndTarget, onChange],
    ),
  });

  const dragging = Haul.useDraggingState();
  const onDragStart = useCallback(
    () => startDrag([{ type: HAUL_TYPE, key: value }]),
    [startDrag, value],
  );

  return (
    <Select.Single
      className={CSS(className, CSS.dropRegion(canDrop(dragging, [value])))}
      value={value}
      onDragStart={onDragStart}
      onDragEnd={endDrag}
      onChange={onChange}
      searcher={client?.channels}
      columns={columns}
      emptyContent={emptyContent}
      tagKey={"name"}
      {...dragProps}
      {...props}
    />
  );
};
