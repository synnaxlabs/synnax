// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  List,
  ListColumn,
  Menu as PMenu,
  Divider,
  ContextMenuMenuProps,
} from "@synnaxlabs/pluto";

import type { Range } from "../store";

import { Menu } from "@/components";

export const rangeListColumns: Array<ListColumn<Range>> = [
  {
    key: "name",
    name: "Name",
  },
  {
    key: "start",
    name: "Start",
    stringer: ({ start }) => new TimeStamp(start).fString("dateTime", "local"),
  },
  {
    key: "end",
    name: "End",
    stringer: ({ start, end }) => {
      const startTS = new TimeStamp(start);
      const endTS = new TimeStamp(end);
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
}: RangesListProps): JSX.Element => {
  const contextMenProps = PMenu.useContextMenu();

  const RangesContextMenu = ({ keys }: ContextMenuMenuProps): JSX.Element => {
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
      <PMenu onChange={handleClick}>
        <PMenu.Item startIcon={<Icon.Edit />} size="small" itemKey="edit">
          Edit Range
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Delete />} size="small" itemKey="remove">
          Remove Range
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Add />} size="small" itemKey="create">
          Create Range
        </PMenu.Item>
        <Divider direction="x" padded />
        <Menu.Item.HardReload />
      </PMenu>
    );
  };

  return (
    <div style={{ flexGrow: 1 }}>
      <PMenu.ContextMenu
        menu={(props) => <RangesContextMenu {...props} />}
        {...contextMenProps}
      >
        <List data={ranges}>
          <List.Selector
            value={selectedRange == null ? [] : [selectedRange.key]}
            onChange={([key]: readonly string[]) => onSelect(key)}
            allowMultiple={false}
          />
          <List.Column.Header columns={rangeListColumns} />
          <List.Core.Virtual
            itemHeight={30}
            style={{ height: "100%", overflowX: "hidden" }}
          >
            {List.Column.Item}
          </List.Core.Virtual>
        </List>
      </PMenu.ContextMenu>
    </div>
  );
};
