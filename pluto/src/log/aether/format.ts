// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { DataType, notation, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

export const formatZ = log.channelEntryZ
  .pick({ notation: true, precision: true, timestampConfig: true })
  .extend({ dataType: z.string().optional() });

/** How a log draws the values of one channel. */
export interface Format extends z.infer<typeof formatZ> {}

const ZERO_FORMAT: Format = formatZ.parse({});

/**
 * Returns the format of each channel, keyed by channel. A channel with a data type and
 * no entry takes the default format.
 */
export const formatsOf = (
  channels: log.ChannelEntry[],
  dataTypes: Record<string, string>,
): Record<string, Format> => {
  const formats: Record<string, Format> = {};
  for (const [key, dataType] of Object.entries(dataTypes))
    formats[key] = { ...ZERO_FORMAT, dataType };
  for (const entry of channels)
    formats[entry.channel] = {
      notation: entry.notation,
      precision: entry.precision,
      timestampConfig: entry.timestampConfig,
      dataType: dataTypes[entry.channel],
    };
  return formats;
};

const INTEGER = /^-?\d+$/;

// A sample shows as it arrives when the channel holds no numbers, its data type has
// not loaded, or a calculated channel changed type after the sample arrived.
const formatNumber = (
  value: string,
  { dataType, precision, notation: valueNotation }: Format,
): string => {
  if (dataType == null) return value;
  const type = new DataType(dataType);
  if (type.isInteger && INTEGER.test(value))
    return notation.stringifyNumber(BigInt(value), precision, valueNotation);
  if (!type.isFloat || value.trim() === "") return value;
  const number = Number(value);
  if (Number.isNaN(number)) return value;
  return notation.stringifyNumber(number, precision, valueNotation);
};

/**
 * Returns the text a log draws for a value of a channel. A value with no format shows
 * as it arrives.
 */
export const textOf = (raw: string, format: Format | undefined): string => {
  if (format == null) return raw;
  if (format.dataType === "timestamp") {
    const { format: timeFormat, timeZone } = format.timestampConfig;
    try {
      return new TimeStamp(BigInt(raw)).toString(timeFormat, timeZone);
    } catch {
      return raw;
    }
  }
  if (format.precision == null && format.notation === "standard") return raw;
  return formatNumber(raw, format);
};
