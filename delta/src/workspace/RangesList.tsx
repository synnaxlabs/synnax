// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { List, Menu as PMenu, Divider } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import type { Range } from "@/workspace/range";

export const rangeListColumns: Array<List.ColumnSpec<string, Range>> = [
  {
    key: "name",
    name: "Name",
  },
  {
    key: "start",
    name: "Start",
    width: 100,
    stringer: (r) => {
      if (r.variant === "dynamic") return `${new TimeSpan(r.span).toString()} ago`;
      return new TimeStamp(r.timeRange.start).fString("dateTime", "local");
    },
  },
  {
    key: "end",
    name: "End",
    stringer: (r) => {
      if (r.variant === "dynamic") return "Now";
      const startTS = new TimeStamp(r.timeRange.start);
      const endTS = new TimeStamp(r.timeRange.end);
      return endTS.fString(
        endTS.span(startTS) < TimeSpan.DAY ? "time" : "dateTime",
        "local"
      );
    },
  },
];

export interface RangesListProps {
  selectedRange?: Range | null;
  onAddOrEdit: (key?: string) => void;
  onSelect: (key: string) => void;
  onRemove: (key: string) => void;
  ranges: Range[];
}

export const RangesList = ({
  ranges,
  selectedRange,
  onAddOrEdit,
  onSelect,
  onRemove,
}: RangesListProps): ReactElement => {
  const contextMenProps = PMenu.useContextMenu();

  const RangesContextMenu = ({ keys }: PMenu.ContextMenuMenuProps): ReactElement => {
    const handleClick = (key: string): void => {
      switch (key) {
        case "create":
          return onAddOrEdit();
        case "edit":
          return onAddOrEdit(keys[0]);
        case "remove":
          return onRemove(keys[0]);
      }
    };
    return (
      <PMenu.Menu onChange={handleClick}>
        <PMenu.Item startIcon={<Icon.Edit />} size="small" itemKey="edit">
          Edit Range
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Delete />} size="small" itemKey="remove">
          Remove Range
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Add />} size="small" itemKey="create">
          Create Range
        </PMenu.Item>
        <Divider.Divider direction="x" padded />
        <Menu.Item.HardReload />
      </PMenu.Menu>
    );
  };

  return (
    <div style={{ flexGrow: 1 }}>
      <PMenu.ContextMenu
        menu={(props) => <RangesContextMenu {...props} />}
        {...contextMenProps}
      >
        <List.List data={ranges}>
          <List.Selector
            value={selectedRange == null ? [] : [selectedRange.key]}
            onChange={([key]: string[]) => onSelect(key)}
            allowMultiple={false}
          />
          <List.Column.Header columns={rangeListColumns} />
          <List.Core.Virtual
            itemHeight={30}
            style={{ height: "100%", overflowX: "hidden" }}
          >
            {List.Column.Item}
          </List.Core.Virtual>
        </List.List>
      </PMenu.ContextMenu>
    </div>
  );
};
