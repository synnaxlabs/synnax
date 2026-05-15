// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@synnaxlabs/x/telem";
import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";
export interface TimeRangeProps
  extends
    Omit<Flex.BoxProps<"div">, "children">,
    Pick<Text.TextProps, "level" | "color" | "weight"> {
  children: telem.CrudeTimeRange;
  displayTZ?: telem.TZInfo;
}

const formatTime = (
  timeRange: telem.CrudeTimeRange,
  displayTZ: telem.TZInfo,
): null | string | [string, string] => {
  const tr = new telem.TimeRange(timeRange).makeValid();
  if (tr.start.equals(telem.TimeStamp.MAX)) return null;
  const startFormat = tr.start.isToday ? "time" : "dateTime";
  let startTime = new telem.TimeStamp(tr.start).toString(startFormat, displayTZ);
  if (tr.start.isToday) startTime = `Today ${startTime}`;
  if (tr.end.equals(telem.TimeStamp.MAX)) {
    if (tr.start.before(telem.TimeStamp.now())) return `Started ${startTime}`;
    return `Starts ${startTime}`;
  }
  const endFormat = tr.end.span(tr.start) < telem.TimeSpan.DAY ? "time" : "dateTime";
  const endTime = new telem.TimeStamp(tr.end).toString(endFormat, displayTZ);
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
