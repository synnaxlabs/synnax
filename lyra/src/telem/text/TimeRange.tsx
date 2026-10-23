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
import { type ReactElement, useEffect, useState } from "react";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface TimeRangeProps
  extends
    Omit<Flex.BoxProps<"div">, "children">,
    Pick<Text.TextProps, "level" | "color" | "weight"> {
  children: CrudeTimeRange;
  displayTimeZone?: TimeZone;
}

const REFRESH_INTERVAL = TimeSpan.MINUTE;

/**
 * @returns the current time, refreshed every minute and at the given instant, so a
 * label that reads the clock catches up when the day turns or a range starts.
 */
const useNow = (next: TimeStamp): TimeStamp => {
  const [now, setNow] = useState(() => TimeStamp.now());
  const nextValue = next.valueOf();
  useEffect(() => {
    const until = new TimeStamp(nextValue);
    let delay = REFRESH_INTERVAL;
    if (until.after(now) && until.span(now).lessThan(delay)) delay = until.span(now);
    const t = setTimeout(() => setNow(TimeStamp.now()), Math.ceil(delay.milliseconds));
    return () => clearTimeout(t);
  }, [now, nextValue]);
  return now;
};

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
    const verb = tr.start.before(now) ? "Started" : "Starts";
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
  const now = useNow(tr.start);
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
