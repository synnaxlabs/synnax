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

import { Align } from "@/align";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Text } from "@/text";

export interface TimeRangeChipProps
  extends Align.SpaceProps<"div">,
    Pick<Text.TextProps, "level" | "shade"> {
  timeRange: CrudeTimeRange;
  showSpan?: boolean;
  labeled?: boolean;
}

export const TimeRangeChip = ({
  timeRange,
  level = "p",
  shade = 9,
  showSpan = false,
  labeled = false,
  ...rest
}: TimeRangeChipProps): ReactElement => {
  const startTS = new TimeStamp(timeRange.start);
  const startFormat = startTS.isToday ? "time" : "dateTime";
  const endTS = new TimeStamp(timeRange.end);
  const isOpen = endTS.equals(TimeStamp.MAX);
  const endFormat = endTS.span(startTS) < TimeSpan.DAY ? "time" : "dateTime";
  const span = startTS.span(endTS);

  let startTime = (
    <Align.Space x align="center" gap="small">
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
    </Align.Space>
  );

  let endTime = (
    <>
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
      {!span.isZero && showSpan && (
        <Text.Text level={level} shade={shade} weight={450}>
          ({startTS.span(endTS).truncate(TimeSpan.MILLISECOND).toString()})
        </Text.Text>
      )}
    </>
  );

  if (labeled) {
    startTime = (
      <Input.Item label="Start" showHelpText={false}>
        {startTime}
      </Input.Item>
    );
    endTime = (
      <Input.Item label="End" showHelpText={false}>
        {endTime}
      </Input.Item>
    );
  }

  const levelVar = CSS.levelSizeVar(level);

  return (
    <Align.Space
      x
      gap="small"
      className={CSS(CSS.B("time-range-chip"))}
      align="center"
      {...rest}
    >
      {startTime}
      <Icon.Arrow.Right
        color="var(--pluto-gray-l9)"
        style={{
          width: levelVar,
          height: levelVar,
        }}
      />
      {endTime}
    </Align.Space>
  );
};
