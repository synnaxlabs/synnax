// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Flex,
  Icon,
  Divider,
  Ranger,
  Text,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/pluto";

interface ListItemProps {
  name: string;
  timeRange?: TimeRange;
}

const ListItem = ({ name, timeRange }: ListItemProps) => (
  <Flex.Box x justify="between">
    <Text.Text>
      <Icon.Range />
      {name}
    </Text.Text>
    {timeRange != null && <Ranger.TimeRangeChip timeRange={timeRange} showSpan />}
  </Flex.Box>
);

const start = TimeStamp.now();

const TIME_RANGES = [
  { name: "Setup", timeRange: start.spanRange(TimeSpan.hours(1)) },
  {
    name: "Test Run",
    timeRange: start.add(TimeSpan.hours(1)).spanRange(TimeSpan.minutes(5)),
  },
  {
    name: "Safing",
    timeRange: start
      .add(TimeSpan.hours(1))
      .add(TimeSpan.minutes(30))
      .spanRange(TimeSpan.minutes(20)),
  },
];

export const ChildRanges = () => (
  <Flex.Box
    x
    style={{
      width: "100vw",
    }}
  >
    <Flex.Box
      y
      style={{
        background: "var(--pluto-gray-l1)",
        padding: "2rem",
        border: "var(--pluto-border-l5)",
        borderRadius: "1rem",
        width: 500,
      }}
    >
      <ListItem name="Test 1" />
      <Flex.Box
        y
        justify="between"
        style={{
          marginLeft: "1rem",
          marginTop: "1rem",
          borderLeft: "var(--pluto-border)",
          padding: "2rem",
        }}
      >
        {TIME_RANGES.map((item, i) => (
          <>
            <ListItem key={item.name} {...item} />
            {i !== TIME_RANGES.length - 1 && <Divider.Divider x />}
          </>
        ))}
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
