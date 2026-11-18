// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, TimeRange } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, Ranger } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { useCreateModal } from "@/platform/range/useCreateModal";

const NONE = "none";
const NEWEST: ranger.Order = { field: "time_range_start", descending: true };

const ParentName = ({ parent }: { parent: ranger.Key }): ReactElement | null => {
  const range = Ranger.useResult({ key: parent }).data;
  if (range == null) return null;
  return (
    <Text.Text level="small" color={9} overflow="fade">
      {range.name}
    </Text.Text>
  );
};

interface RowProps {
  range: ranger.Range;
}

const Row = ({ range }: RowProps): ReactElement => (
  <>
    <Text.Text overflow="fade" className={CSS.BE("range-menu", "name")}>
      {range.name}
    </Text.Text>
    {range.parent != null && <ParentName parent={range.parent} />}
    <Telem.Text.TimeRange
      level="small"
      color={9}
      className={CSS.BE("range-menu", "time")}
    >
      {range.timeRange}
    </Telem.Text.TimeRange>
  </>
);

const Item = (props: List.ItemRenderProps<ranger.Key>): ReactElement | null => {
  const range = List.useItem<ranger.Key, ranger.Range>(props.itemKey);
  if (range == null) return null;
  return (
    <Select.Item {...props} gap="small">
      <Row range={range} />
    </Select.Item>
  );
};

const item = Component.renderProp(Item);

const Current = ({ parent }: { parent: ranger.Key }): ReactElement | null => {
  const range = Ranger.useResult({ key: parent }).data;
  if (range == null) return null;
  return (
    <Select.Item<ranger.Key> itemKey={parent} textValue={range.name} gap="small">
      <Row range={range} />
    </Select.Item>
  );
};

export interface ParentProps {
  ranges: ranger.Range[];
}

/**
 * Moves the ranges under another range, or to the top level. A new range made here
 * covers the ranges and takes their shared parent, and the ranges move inside it.
 */
export const Parent = ({ ranges }: ParentProps): ReactElement => {
  const openCreate = useCreateModal();
  const { capture } = Analytics.use();
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const { update } = Ranger.useSetParent();
  const first = ranges[0]?.parent ?? null;
  const shared = ranges.every((r) => (r.parent ?? null) === first);
  const current = shared ? first : null;
  const keys = ranges.map((r) => r.key);
  const nested = Ranger.useResultList({ where: { ancestors: keys } });
  // A range cannot hold itself or a range nested in it, and the current parent leads
  // the list.
  const hidden = [...keys, ...(nested.data ?? []).map((r) => r.key)];
  if (current != null) hidden.push(current);
  // The Core returns nested ranges in no fixed order, and a new order is a new query.
  hidden.sort();
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (c) => c.ranges,
    {
      name: "ranges",
      // The list waits for the nested ranges, so it never shows one of them.
      query:
        nested.data == null
          ? null
          : { order: NEWEST, where: { not: { key: { in: hidden } } } },
    },
  );
  const move = (parent: ranger.Key | null) =>
    ranges.forEach((r) => update({ key: r.key, parent }));
  const handleChange = (next: ranger.Key | null) =>
    move(next == null || next === NONE ? null : next);
  const { update: create } = Ranger.useCreate({
    afterSuccess: ({ data }) => {
      capture("range_created", {});
      move(data.key);
    },
  });
  const handleCreate = (name: string) => {
    const timeRange = TimeRange.max(...ranges.map((r) => r.timeRange));
    if (name !== "") return create({ name, timeRange, parent: current ?? undefined });
    void openCreate({ timeRange: timeRange.numeric, parent: current ?? "" }).then(
      (key) => {
        if (key != null) move(key);
      },
    );
  };
  return (
    <Menu.Sub itemKey="parent" textValue="Parent">
      <Menu.SubTrigger>
        <Ranger.ParentIcon />
        Parent
      </Menu.SubTrigger>
      <Select.Frame<ranger.Key, ranger.Range | undefined>
        value={shared ? (first ?? NONE) : undefined}
        onChange={handleChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
        closeDialogOnSelect
      >
        <Select.Dialog
          surface="menu"
          aria-label="Parent"
          className={CSS.BE("range-menu", "parent")}
        >
          <Select.Body<ranger.Key>
            resourceName="range"
            onSearch={search}
            status={nested.variant === "success" ? status : nested.status}
            fixedItems={current != null && <Current parent={current} />}
            footer={
              <>
                <Menu.Divider />
                {canCreate && (
                  <Select.CreateItem<ranger.Key, ranger.Range>
                    resourceName="range"
                    onCreate={handleCreate}
                  />
                )}
                <Select.Item<ranger.Key> itemKey={NONE} textValue="No parent">
                  <Icon.None />
                  No parent
                </Select.Item>
              </>
            }
          >
            {item}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    </Menu.Sub>
  );
};
