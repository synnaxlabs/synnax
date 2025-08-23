// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/ranger/TimeRangeChip.css";

import { type CrudeTimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface TimeRangeChipProps
  extends Flex.BoxProps<"div">,
    Pick<Text.TextProps, "level" | "color"> {
  timeRange: CrudeTimeRange;
  showSpan?: boolean;
}

export const TimeRangeChip = ({
  timeRange,
  level = "p",
  color = 9,
  showSpan = false,
  ...rest
}: TimeRangeChipProps): ReactElement => {
  const startTS = new TimeStamp(timeRange.start);
  const startFormat = startTS.isToday ? "time" : "dateTime";
  const endTS = new TimeStamp(timeRange.end);
  const isOpen = endTS.equals(TimeStamp.MAX);
  const endFormat = endTS.span(startTS) < TimeSpan.DAY ? "time" : "dateTime";
  const span = startTS.span(endTS);

  const startTime = (
    <Text.Text x align="center" gap="small">
      {startTS.isToday && (
        <Text.Text level={level} color={color} weight={450}>
          Today
        </Text.Text>
      )}
      <Text.DateTime
        el="span"
        level={level}
        displayTZ="local"
        format={startFormat}
        color={color}
        weight={450}
      >
        {startTS}
      </Text.DateTime>
    </Text.Text>
  );

  const endTime = (
    <>
      {isOpen ? (
        <Text.Text level={level} el="span">
          Now
        </Text.Text>
      ) : (
        <Text.DateTime
          level={level}
          el="span"
          displayTZ="local"
          format={endFormat}
          color={color}
          weight={450}
        >
          {endTS}
        </Text.DateTime>
      )}
      {!span.isZero && showSpan && (
        <Text.Text level={level} color={color} weight={450} el="span">
          ({startTS.span(endTS).truncate(TimeSpan.MILLISECOND).toString()})
        </Text.Text>
      )}
    </>
  );

  return (
    <Text.Text
      x
      gap="small"
      className={CSS(CSS.B("time-range-chip"))}
      align="center"
      level={level}
      color={color}
      {...rest}
    >
      {startTime}
      <Icon.Arrow.Right color={9} />
      {endTime}
    </Text.Text>
  );
};
