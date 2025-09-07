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
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Text } from "@/text";

export interface TimeRangeChipProps
  extends Flex.BoxProps<"div">,
    Pick<Text.TextProps, "level" | "color"> {
  timeRange: CrudeTimeRange;
  showSpan?: boolean;
  labeled?: boolean;
  collapseZero?: boolean;
  offsetFrom?: TimeStamp;
  showAgo?: boolean;
  variant?: "text" | "outlined";
}

export const TimeRangeChip = ({
  timeRange,
  level = "p",
  color = 9,
  showSpan = false,
  labeled = false,
  collapseZero = false,
  offsetFrom,
  showAgo = false,
  variant = "text",
  ...rest
}: TimeRangeChipProps): ReactElement => {
  const startTS = new TimeStamp(timeRange.start);
  const startFormat = startTS.isToday ? "time" : "dateTime";
  const endTS = new TimeStamp(timeRange.end);
  const isOpen = endTS.equals(TimeStamp.MAX);
  const isZero = startTS.equals(endTS);
  const endFormat = endTS.span(startTS) < TimeSpan.DAY ? "time" : "dateTime";
  const span = startTS.span(endTS);

  let startTime = (
    <Flex.Box x align="center" gap="small">
      {startTS.isToday && (
        <Text.Text level={level} color={color} weight={450}>
          Today
        </Text.Text>
      )}
      <Text.DateTime
        level={level}
        displayTZ="local"
        format={startFormat}
        color={color}
        weight={450}
      >
        {startTS}
      </Text.DateTime>
    </Flex.Box>
  );

  let endTime: ReactElement | null = (
    <>
      {isOpen ? (
        <Text.Text level={level}>Now</Text.Text>
      ) : (
        <Text.DateTime
          level={level}
          displayTZ="local"
          format={endFormat}
          color={color}
          weight={450}
        >
          {endTS}
        </Text.DateTime>
      )}
      {!span.isZero && showSpan && (
        <Text.Text level={level} color={color} weight={450}>
          ({span.truncate(TimeSpan.MILLISECOND).toString()})
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

  let arrow: ReactElement | null = (
    <Icon.Arrow.Right
      color={9}
      style={{
        width: "1em",
        height: "1em",
      }}
    />
  );

  if (collapseZero && isZero) {
    endTime = null;
    arrow = null;
  }

  let offset: ReactElement | null = null;
  if (offsetFrom != null) {
    let offsetSpan = offsetFrom.span(startTS);
    let character = "+";
    if (offsetSpan.lessThan(0)) {
      character = "-";
      offsetSpan = offsetSpan.mult(-1);
    }
    offset = (
      <Text.Text level={level} color={color} weight={450}>
        T{character} {offsetSpan.truncate(TimeSpan.MILLISECOND).toString()}
      </Text.Text>
    );
  }

  let ago: ReactElement | null = null;
  if (showAgo) {
    let agoSpan = startTS.span(TimeStamp.now());
    if (agoSpan.lessThan(0)) agoSpan = agoSpan.mult(-1);

    ago = (
      <Text.Text level={level} color={color} weight={450}>
        {agoSpan.truncate(TimeSpan.MINUTE).toString()} ago
      </Text.Text>
    );
  }

  return (
    <Flex.Box
      x
      gap="small"
      className={CSS(CSS.B("time-range-chip"), CSS.M(variant))}
      align="center"
      {...rest}
    >
      {startTime}
      {arrow}
      {endTime}
      {offset && (
        <>
          <Divider.Divider y />
          {offset}
        </>
      )}
      {ago && (
        <>
          <Divider.Divider y />
          {ago}
        </>
      )}
    </Flex.Box>
  );
};
