// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/client";

import { Text } from "./Text";
import type { TextProps } from "./Text";

export interface TextDateTimeProps extends Omit<TextProps, "children"> {
  children: Date | TimeStamp;
  format?: TimeDisplayFormat;
}

export type TimeDisplayFormat = "dateShort" | "dateTimeShort" | "timeShort";

export const TextDateTime = ({
  format = "dateTimeShort",
  children,
  ...props
}: TextDateTimeProps): JSX.Element => {
  if (!(children instanceof Date)) children = children.date();
  const formatter = formatFunctions[format];
  return <Text {...props}>{formatter(children)}</Text>;
};

type FormatF = (date: Date) => string;

/**
 * Formats a date into a short, easily readable string in military time.
 *
 * @param date - The date to format.
 * @returns - The formatted date string. Example: "Jan 1 14:00:00"
 */
export const shortDateTimeString: FormatF = (date) =>
  shortDateString(date) + " " + shortTimeString(date);

export const shortDateString: FormatF = (date) => {
  const month = date.toLocaleString("default", { month: "short" });
  const day = date.getDate();
  return `${month} ${day}`;
};

export const shortTimeString: FormatF = (date) => {
  const hours = date.getHours();
  const minutes = date.getMinutes();
  const seconds = date.getSeconds();
  return `${hours}:${minutes}:${seconds}`;
};

const formatFunctions: Record<TimeDisplayFormat, FormatF> = {
  dateShort: shortDateString,
  dateTimeShort: shortDateTimeString,
  timeShort: shortTimeString,
};
