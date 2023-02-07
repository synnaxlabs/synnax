// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Text, List, ListColumn, Menu, Divider } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { setActiveRange, useSelectRanges, useSelectRange } from "../store";
import type { Range } from "../store";
import { Icon } from "@/components/Icon";

export const rangeListColumns: Array<ListColumn<Range>> = [
  {
    key: "name",
    name: "Name",
  },
  {
    key: "start",
    name: "Start",
    render: ({ entry: { start }, style }) => (
      <Text.DateTime level="p" style={style}>
        {start}
      </Text.DateTime>
    ),
  },
  {
    key: "end",
    name: "End",
    render: ({ entry: { start, end }, style }) => {
      const startTS = new TimeStamp(start);
      const endTS = new TimeStamp(end);
      return (
        <Text.DateTime
          level="p"
          style={style}
          format={endTS.span(startTS) < TimeSpan.DAY ? "time" : "dateTime"}
        >
          {endTS}
        </Text.DateTime>
      );
    },
  },
];

export const RangesList = (): JSX.Element => {
  const ranges = useSelectRanges();
  const selectedRange = useSelectRange();
  const dispatch = useDispatch();
  return (
    <div style={{ flexGrow: 1 }}>
      <Menu.Context menu={(props) => <RangesContextMenu />}>
        <List data={ranges}>
          <List.Selector
            value={selectedRange == null ? [] : [selectedRange.key]}
            onChange={([key]: readonly string[]) => dispatch(setActiveRange(key ?? null))}
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
      </Menu.Context>
    </div>
  );
};

export const RangesContextMenu = (): JSX.Element => {
  return (
    <Menu>
      <Menu.Item startIcon={<Icon.Edit />} size="small">Edit Range</Menu.Item>
      <Menu.Item startIcon={<Icon.Delete />} size="small">Remove Range</Menu.Item>
      <Menu.Item startIcon={<Icon.Create />} size="small">Create Range</Menu.Item>
      <Divider direction="x" />
      <Menu.Item startIcon={<Icon.Refresh />} size="small">Hard Reload</Menu.Item>
    </Menu>
  )
}
    

