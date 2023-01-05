// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Text, List } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { setActiveRange, useSelectRanges, useSelectRange } from "../store";
import type { Range } from "../store";

export const RangesList = (): JSX.Element => {
  const ranges = useSelectRanges();
  const selectedRange = useSelectRange();
  const dispatch = useDispatch();
  return (
    <List data={ranges}>
      <List.Selector
        value={selectedRange == null ? [] : [selectedRange.key]}
        onChange={([key]: readonly string[]) => dispatch(setActiveRange(key ?? null))}
        allowMultiple={false}
      />
      <List.Column.Header<Range>
        columns={[
          {
            key: "name",
            label: "Name",
          },
          {
            key: "start",
            label: "Start",
            render: ({ entry: { start }, style }) => {
              return (
                <Text.DateTime level="p" style={style}>
                  {start}
                </Text.DateTime>
              );
            },
          },
          {
            key: "end",
            label: "End",
            render: ({ entry: { start, end }, style }) => {
              const startTS = new TimeStamp(start);
              const endTS = new TimeStamp(end);
              return (
                <Text.DateTime
                  level="p"
                  style={style}
                  format={
                    endTS.span(startTS) < TimeSpan.Day ? "shortTime" : "shortDateTime"
                  }
                >
                  {endTS.valueOf()}
                </Text.DateTime>
              );
            },
          },
        ]}
      />
      <List.Core.Virtual itemHeight={30} style={{ height: "100%" }}>
        {List.Column.Item}
      </List.Core.Virtual>
    </List>
  );
};
