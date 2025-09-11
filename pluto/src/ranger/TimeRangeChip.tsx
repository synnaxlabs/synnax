// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/ranger/TimeRangeChip.css";

import { type CrudeTimeRange, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface TimeRangeChipProps
  extends Flex.BoxProps<"div">,
    Pick<Text.TextProps, "level" | "color"> {
  timeRange: CrudeTimeRange;
  variant?: "text" | "outlined";
}

export const TimeRangeChip = ({
  timeRange,
  level = "p",
  color = 9,
  variant = "text",
  ...rest
}: TimeRangeChipProps): ReactElement | null => {
  const tr = new TimeRange(timeRange).makeValid();
  const startIsUnknown = tr.start.equals(TimeStamp.MAX);
  const startIsToday = tr.start.isToday;
  const startFormat = startIsToday ? "time" : "dateTime";
  const startTime = startIsUnknown ? (
    <Text.Text level={level} color={color} weight={450}>
      ?
    </Text.Text>
  ) : (
    <Flex.Box x align="center" gap="small">
      {startIsToday && (
        <Text.Text level={level} color={color} weight={450}>
          Today
        </Text.Text>
      )}
      <Text.DateTime level={level} format={startFormat} color={color} weight={450}>
        {tr.start}
      </Text.DateTime>
    </Flex.Box>
  );
  const endIsUnknown = tr.end.equals(TimeStamp.MAX);
  const endFormat = tr.end.span(tr.start) < TimeSpan.DAY ? "time" : "dateTime";
  const endTime = (
    <>
      {endIsUnknown ? (
        <Text.Text level={level} color={color} weight={450}>
          ?
        </Text.Text>
      ) : (
        <Text.DateTime
          level={level}
          displayTZ="local"
          format={endFormat}
          color={color}
          weight={450}
        >
          {tr.end}
        </Text.DateTime>
      )}
    </>
  );
  return (
    <Flex.Box
      x
      gap="small"
      className={CSS(CSS.B("time-range-chip"), CSS.M(variant))}
      align="center"
      {...rest}
    >
      {startTime}
      <Icon.Arrow.Right color={9} style={{ height: "1em", width: "1em" }} />
      {endTime}
    </Flex.Box>
  );
};
