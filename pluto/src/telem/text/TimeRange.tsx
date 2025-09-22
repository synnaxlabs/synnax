// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Icon } from "@/icon";
import { TimeSpan as TimeSpanText } from "@/telem/text/TimeSpan";
import {
  TimeStamp as TimeStampText,
  type TimeStampProps as TimeStampTextProps,
} from "@/telem/text/TimeStamp";
import { Text } from "@/text";

export interface TimeRange extends Omit<TimeStampTextProps, "children" | "format"> {
  showSpan?: boolean;
  children: CrudeTimeRange;
}

export const TimeRange = ({
  children: timeRange,
  level = "p",
  color = 9,
  showSpan = false,
  suppliedTZ,
  displayTZ,
  ...rest
}: TimeRange): ReactElement => {
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
      <TimeStampText
        el="span"
        level={level}
        displayTZ={displayTZ}
        suppliedTZ={suppliedTZ}
        format={startFormat}
        color={color}
        weight={450}
      >
        {startTS}
      </TimeStampText>
    </Text.Text>
  );

  const endTime = (
    <>
      {isOpen ? (
        <Text.Text level={level} el="span">
          Now
        </Text.Text>
      ) : (
        <TimeStampText
          level={level}
          el="span"
          displayTZ={displayTZ}
          suppliedTZ={suppliedTZ}
          format={endFormat}
          color={color}
          weight={450}
        >
          {endTS}
        </TimeStampText>
      )}
      {!span.isZero && showSpan && (
        <TimeSpanText level={level} el="span" color={color} weight={450}>
          {startTS.span(endTS).truncate(TimeSpan.MILLISECOND)}
        </TimeSpanText>
      )}
    </>
  );

  return (
    <Text.Text x gap="small" align="center" level={level} color={color} {...rest}>
      {startTime}
      <Icon.Arrow.Right color={9} />
      {endTime}
    </Text.Text>
  );
};
