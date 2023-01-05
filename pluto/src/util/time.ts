// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const maybePad = (num: number, count = 2): string => {
  const s = num.toString();
  return s.length >= count ? s : "0".repeat(count - s.length) + s;
};

type NanoStringF = (nano: number) => string;
type StringNanoF = (str: string) => number;

export const isoStringNano: StringNanoF = (str): number => {
  if (str.length === 0) return 0;
  return new Date(str).getTime() * 1e6;
};

export const timeStringNano: StringNanoF = (str) => {
  // take a string of the format HH:MM:SS and convert it to a nanosecond timestamp
  if (str.length === 0) return 0;
  const p = str.split(":");
  const d = new Date();
  d.setHours(parseInt(p[0], 10));
  d.setMinutes(parseInt(p[1], 10));
  d.setSeconds(parseInt(p[2], 10));
  return d.getTime() * 1e6;
};

export const nanoTimeString: NanoStringF = (nano) => {
  // take a nanosecond timestamp and convert it to a HH:MM:SS string
  const d = nanoDate(nano);
  return `${maybePad(d.getHours())}:${maybePad(d.getMinutes())}:${maybePad(
    d.getSeconds()
  )}`;
};

export const nanoISOString: NanoStringF = (nano) => nanoDate(nano).toISOString();

/**
 * Formats a date into a short, easily readable string in military time.
 *
 * @param nano - The date to format.
 * @returns - The formatted date string. Example: "Jan 1 14:00:00"
 */
export const nanoShortDateTimeString: NanoStringF = (nano) =>
  nanoShortDateString(nano) + " " + nanoTimeString(nano);

export const nanoShortDateString: NanoStringF = (nano) => {
  const d = nanoDate(nano);
  const month = d.toLocaleString("default", { month: "short" });
  const day = d.getDate();
  return `${month} ${maybePad(day)}`;
};

export const shortDateISOString: NanoStringF = (nano) => {
  const d = nanoDate(nano);
  return `${maybePad(d.getFullYear(), 4)}-${maybePad(d.getMonth() + 1)}-${maybePad(
    d.getDate()
  )}`;
};

export const isoStringShortDate: StringNanoF = (str) => {
  if (str.length === 0) return 0;
  const p = str.split("-");
  const d = new Date();
  d.setFullYear(parseInt(p[0], 10));
  d.setMonth(parseInt(p[1], 10) - 1);
  d.setDate(parseInt(p[2], 10));
  return d.getTime() * 1e6;
};

export const nanoDate = (nano: number): Date => new Date((nano ?? 0) / 1e6);

export type TimeStringFormatter = "shortTime" | "shortDate" | "shortDateTime" | "iso";

export const timeStringFormatters: Record<TimeStringFormatter, NanoStringF> = {
  shortTime: nanoTimeString,
  shortDate: nanoShortDateString,
  shortDateTime: nanoShortDateTimeString,
  iso: nanoISOString,
};
