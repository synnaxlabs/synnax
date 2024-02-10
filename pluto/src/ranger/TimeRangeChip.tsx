// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { type CrudeTimeRange, TimeStamp, TimeSpan } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Text } from "@/text";

import "@/ranger/TimeRangeChip.css";

export interface TimeRangeChipProps {
  timeRange: CrudeTimeRange;
}

export const TimeRangeChip = ({ timeRange }: TimeRangeChipProps): ReactElement => {
  const startTS = new TimeStamp(timeRange.start);
  const startFormat = startTS.isToday ? "time" : "dateTime";
  const endTS = new TimeStamp(timeRange.end);
  const isOpen = endTS.equals(TimeStamp.MAX);
  const endFormat = endTS.span(startTS) < TimeSpan.DAY ? "time" : "dateTime";

  return (
    <Align.Space
      direction="x"
      size="small"
      className={CSS(CSS.B("time-range-chip"))}
      align="center"
    >
      {startTS.isToday && (
        <Text.Text level="p" shade={7} weight={450}>
          Today
        </Text.Text>
      )}
      <Text.DateTime
        level="p"
        displayTZ="local"
        format={startFormat}
        shade={7}
        weight={450}
      >
        {startTS}
      </Text.DateTime>
      <Icon.Arrow.Right color="var(--pluto-text-color)" />
      {isOpen ? (
        <Text.Text level="p">Now</Text.Text>
      ) : (
        <Text.DateTime
          level="p"
          displayTZ="local"
          format={endFormat}
          shade={7}
          weight={450}
        >
          {endTS}
        </Text.DateTime>
      )}
    </Align.Space>
  );
};
