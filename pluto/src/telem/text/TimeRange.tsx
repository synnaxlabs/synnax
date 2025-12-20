// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CrudeTimeRange,
  TimeRange as XTimeRange,
  TimeSpan,
  TimeStamp,
  type TZInfo,
} from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface TimeRangeProps
  extends
    Omit<Flex.BoxProps<"div">, "children">,
    Pick<Text.TextProps, "level" | "color" | "weight"> {
  children: CrudeTimeRange;
  displayTZ?: TZInfo;
}

const formatTime = (
  timeRange: CrudeTimeRange,
  displayTZ: TZInfo,
): null | string | [string, string] => {
  const tr = new XTimeRange(timeRange).makeValid();
  if (tr.start.equals(TimeStamp.MAX)) return null;
  const startFormat = tr.start.isToday ? "time" : "dateTime";
  let startTime = new TimeStamp(tr.start).toString(startFormat, displayTZ);
  if (tr.start.isToday) startTime = `Today ${startTime}`;
  if (tr.end.equals(TimeStamp.MAX)) {
    if (tr.start.before(TimeStamp.now())) return `Started ${startTime}`;
    return `Starts ${startTime}`;
  }
  const endFormat = tr.end.span(tr.start) < TimeSpan.DAY ? "time" : "dateTime";
  const endTime = new TimeStamp(tr.end).toString(endFormat, displayTZ);
  return [startTime, endTime];
};

export const TimeRange = ({
  children,
  level = "p",
  color = 9,
  displayTZ = "local",
  weight = 450,
  ...rest
}: TimeRangeProps): ReactElement | null => {
  const formattedTime = formatTime(children, displayTZ);
  if (formattedTime == null) return null;
  return (
    <Flex.Box x gap="small" align="center" {...rest}>
      <Text.Text level={level} color={color} weight={weight} gap="tiny">
        {typeof formattedTime === "string" ? (
          formattedTime
        ) : (
          <>
            {formattedTime[0]}
            <Icon.Arrow.Right color={9} size="1em" />
            {formattedTime[1]}
          </>
        )}
      </Text.Text>
    </Flex.Box>
  );
};
