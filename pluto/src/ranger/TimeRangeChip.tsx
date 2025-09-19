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

const formatTime = (timeRange: CrudeTimeRange): null | string | [string, string] => {
  const tr = new TimeRange(timeRange).makeValid();
  if (tr.start.equals(TimeStamp.MAX)) return null;
  const startFormat = tr.start.isToday ? "time" : "dateTime";
  let startTime = new TimeStamp(tr.start).fString(startFormat, "local");
  if (tr.start.isToday) startTime = `Today ${startTime}`;
  if (tr.end.equals(TimeStamp.MAX)) {
    if (tr.start.before(TimeStamp.now())) return `Started ${startTime}`;
    return `Starts ${startTime}`;
  }
  const endFormat = tr.end.span(tr.start) < TimeSpan.DAY ? "time" : "dateTime";
  const endTime = new TimeStamp(tr.end).fString(endFormat, "local");
  return [startTime, endTime];
};

export const TimeRangeChip = ({
  timeRange,
  level = "p",
  color = 9,
  variant = "text",
  ...rest
}: TimeRangeChipProps): ReactElement | null => {
  const formattedTime = formatTime(timeRange);
  if (formattedTime == null) return null;
  return (
    <Flex.Box
      x
      gap="small"
      className={CSS(CSS.B("time-range-chip"), CSS.M(variant))}
      align="center"
      {...rest}
    >
      <Text.Text level={level} color={color} weight={450} gap="tiny">
        {typeof formattedTime === "string" ? (
          formattedTime
        ) : (
          <>
            {formattedTime[0]}
            <Icon.Arrow.Right color={9} style={{ height: "1em", width: "1em" }} />
            {formattedTime[1]}
          </>
        )}
      </Text.Text>
    </Flex.Box>
  );
};
