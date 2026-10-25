// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CrudeTimeStamp,
  TimeSpan as XTimeSpan,
  type TimeSpanStringFormat,
  type TimeStamp as XTimeStamp,
} from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Generic } from "@/generic";
import { useNow } from "@/hooks";
import { TimeSpan, type TimeSpanProps } from "@/telem/text/TimeSpan";

export type TimeSpanSinceProps<E extends Generic.ElementType = "p"> = Omit<
  TimeSpanProps<E>,
  "children"
> & {
  children: CrudeTimeStamp;
  format?: TimeSpanStringFormat;
};

const nextSecond = (now: XTimeStamp): XTimeStamp => now.add(XTimeSpan.SECOND);

/** @returns the time elapsed since the stamp, refreshed every second. */
export const useTimeSpanSince = (stamp: CrudeTimeStamp): XTimeSpan =>
  useNow(nextSecond).span(stamp);

export const TimeSpanSince = <E extends Generic.ElementType = "p">({
  children,
  ...rest
}: TimeSpanSinceProps<E>): ReactElement => {
  const span = useTimeSpanSince(children);
  return <TimeSpan {...rest}>{span}</TimeSpan>;
};
