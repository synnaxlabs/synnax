// Copyright 2026 Synnax Labs, Inc.
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
  type TimeZone,
} from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { useNow } from "@/hooks";
import { Icon } from "@/icon";
import { nextMidnight } from "@/telem/text/format";
import { Text } from "@/text";

export interface TimeRangeProps
  extends
    Omit<Flex.BoxProps<"div">, "children">,
    Pick<Text.TextProps, "level" | "color" | "weight"> {
  children: CrudeTimeRange;
  displayTimeZone?: TimeZone;
}

const formatTime = (
  tr: XTimeRange,
  now: TimeStamp,
  displayTimeZone: TimeZone,
): null | string | [string, string] => {
  if (tr.start.equals(TimeStamp.MAX)) return null;
  const today = tr.start.isSameDay(now, displayTimeZone);
  const startFormat = today ? "time" : "dateTime";
  const startTime = new TimeStamp(tr.start).toString(startFormat, displayTimeZone);
  if (tr.end.equals(TimeStamp.MAX)) {
    const verb = tr.start.beforeEq(now) ? "Started" : "Starts";
    return today ? `${verb} today ${startTime}` : `${verb} ${startTime}`;
  }
  const endFormat = tr.end.span(tr.start) < TimeSpan.DAY ? "time" : "dateTime";
  const endTime = new TimeStamp(tr.end).toString(endFormat, displayTimeZone);
  return [today ? `Today ${startTime}` : startTime, endTime];
};

export const TimeRange = ({
  children,
  level = "p",
  color = 9,
  displayTimeZone = "local",
  weight = 450,
  ...rest
}: TimeRangeProps): ReactElement | null => {
  const tr = new XTimeRange(children).makeValid();
  // The label changes when the range starts and when the day turns.
  const now = useNow((now) => {
    const midnight = nextMidnight(now, displayTimeZone);
    return tr.start.after(now) ? TimeStamp.min(tr.start, midnight) : midnight;
  });
  const formattedTime = formatTime(tr, now, displayTimeZone);
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
