// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Icon, type IconProps } from "@synnaxlabs/media";
import { DataType, nullToArr, toArray, unique } from "@synnaxlabs/x";
import {
  type DragEvent,
  type FC,
  type ReactElement,
  useCallback,
  useId,
  useMemo,
} from "react";

import { useActiveRange, useAliases } from "@/channel/AliasProvider";
import { HAUL_TYPE } from "@/channel/types";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { type DraggingState } from "@/haul/Haul";
import { type Icon as PIcon } from "@/icon";
import { type List } from "@/list";
import { useMemoDeepEqualProps } from "@/memo";
import { Select } from "@/select";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

const channelColumns: Array<List.ColumnSpec<channel.Key, channel.Payload>> = [
  { key: "name", name: "Name" },
  { key: "alias", name: "Alias" },
  { key: "rate", name: "Rate" },
  { key: "dataType", name: "Data Type" },
  { key: "index", name: "Index" },
  { key: "key", name: "Key" },
  { key: "isIndex", name: "Is Index" },
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
  searchOptions?: channel.RetrieveOptions;
}

const DEFAULT_FILTER = ["name", "alias"];

const useColumns = (
  filter: string[],
): Array<List.ColumnSpec<channel.Key, channel.Payload>> => {
  const aliases = useAliases();
  return useMemo(() => {
    if (filter.length === 0) return channelColumns;
    return channelColumns.filter((column) => filter.includes(column.key));
  }, [filter, aliases]);
};

export const resolveIcon = (ch?: channel.Payload): FC<IconProps> => {
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

const renderTag = ({
  key,
  ...props
}: Select.MultipleTagProps<channel.Key, channel.Payload>): ReactElement => {
  const Icon = resolveIcon(props.entry);
  return <Select.MultipleTag key={key} icon={<Icon />} {...props} />;
};

export const SelectMultiple = ({
  columns: filter = DEFAULT_FILTER,
  onChange,
  className,
  value,
  searchOptions,
  ...props
}: SelectMultipleProps): ReactElement => {
  const client = Synnax.use();
  const aliases = useAliases();
  const columns = useColumns(filter);
  const activeRange = useActiveRange();
  const memoSearchOptions = useMemoDeepEqualProps(searchOptions);
  const searcher = useMemo(
    () =>
      client?.channels.newSearcherWithOptions({
        rangeKey: activeRange,
        internal: false,
        ...memoSearchOptions,
      }),
    [client, activeRange, memoSearchOptions],
  );
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
    type: "Channel.SelectMultiple",
    canDrop: useCallback((hauled) => canDrop(hauled, toArray(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const dropped = Haul.filterByType(HAUL_TYPE, items);
        if (dropped.length === 0) return [];
        const v = unique.unique([
          ...toArray(value),
          ...(dropped.map((c) => c.key) as channel.Keys),
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
    (_: DragEvent<HTMLDivElement>, key: channel.Key) =>
      startDrag([{ key, type: HAUL_TYPE }], handleSuccessfulDrop),
    [startDrag, handleSuccessfulDrop],
  );

  const entryRenderKey = useCallback(
    (e: channel.Payload) => aliases[e.key] ?? e.name,
    [aliases],
  );

  return (
    <Select.Multiple
      className={CSS(className, CSS.dropRegion(canDrop(dragging, toArray(value))))}
      value={value}
      onTagDragStart={onDragStart}
      onTagDragEnd={endDrag}
      searcher={searcher}
      onChange={onChange}
      columns={columns}
      emptyContent={emptyContent}
      entryRenderKey={entryRenderKey}
      renderTag={renderTag}
      {...dropProps}
      {...props}
    />
  );
};

export interface SelectSingleProps
  extends Omit<Select.SingleProps<channel.Key, channel.Payload>, "columns"> {
  columns?: string[];
  searchOptions?: channel.RetrieveOptions;
}

export const SelectSingle = ({
  columns: filter = DEFAULT_FILTER,
  onChange,
  value,
  className,
  data,
  searchOptions,
  ...props
}: SelectSingleProps): ReactElement => {
  const client = Synnax.use();
  const aliases = useAliases();
  const columns = useColumns(filter);
  const activeRange = useActiveRange();
  const memoSearchOptions = useMemoDeepEqualProps(searchOptions);
  const searcher = useMemo(() => {
    if (data != null && data.length > 0) return undefined;
    return client?.channels.newSearcherWithOptions({
      rangeKey: activeRange,
      internal: false,
      ...memoSearchOptions,
    });
  }, [client, activeRange, data?.length, memoSearchOptions]);

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
    canDrop: useCallback((hauled) => canDrop(hauled, nullToArr(value)), [value]),
    onDrop: useCallback(
      ({ items }) => {
        const ch = Haul.filterByType(HAUL_TYPE, items);
        if (ch.length === 0) return [];
        onChange(ch[0].key as channel.Key, {
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

  const entryRenderKey = useCallback(
    (e: channel.Payload) => aliases[e.key] ?? e.name,
    [aliases],
  );

  return (
    <Select.Single
      data={data}
      className={CSS(className, CSS.dropRegion(canDrop(dragging, nullToArr(value))))}
      value={value}
      onDragStart={onDragStart}
      onDragEnd={endDrag}
      onChange={onChange}
      searcher={searcher}
      columns={columns}
      emptyContent={emptyContent}
      entryRenderKey={entryRenderKey}
      {...dragProps}
      {...props}
    />
  );
};
