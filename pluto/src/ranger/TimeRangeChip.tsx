// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/ranger/TimeRangeChip.css";

import { Icon } from "@synnaxlabs/media";
import { type CrudeTimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Text } from "@/text";

export interface TimeRangeChipProps
  extends Align.SpaceProps<"div">,
    Pick<Text.TextProps, "level" | "shade"> {
  timeRange: CrudeTimeRange;
}

export const TimeRangeChip = ({
  timeRange,
  level = "p",
  shade = 7,
  ...props
}: TimeRangeChipProps): ReactElement => {
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
      {...props}
    >
      {startTS.isToday && (
        <Text.Text level={level} shade={shade} weight={450}>
          Today
        </Text.Text>
      )}
      <Text.DateTime
        level={level}
        displayTZ="local"
        format={startFormat}
        shade={shade}
        weight={450}
      >
        {startTS}
      </Text.DateTime>
      <Icon.Arrow.Right color="var(--pluto-text-color)" />
      {isOpen ? (
        <Text.Text level={level}>Now</Text.Text>
      ) : (
        <Text.DateTime
          level={level}
          displayTZ="local"
          format={endFormat}
          shade={shade}
          weight={450}
        >
          {endTS}
        </Text.DateTime>
      )}
    </Align.Space>
  );
};
