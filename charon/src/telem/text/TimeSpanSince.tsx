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
  type TimeSpan as XTimeSpan,
  type TimeSpanStringFormat,
  TimeStamp as XTimeStamp,
} from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { type Generic } from "@/generic";
import { TimeSpan, type TimeSpanProps } from "@/telem/text/TimeSpan";

export type TimeSpanSinceProps<E extends Generic.ElementType = "p"> = Omit<
  TimeSpanProps<E>,
  "children"
> & {
  children: CrudeTimeStamp;
  format?: TimeSpanStringFormat;
};

export const useTimeSpanSince = (stamp: CrudeTimeStamp): XTimeSpan => {
  const [, setCounter] = useState(0);
  useEffect(() => {
    const i = setInterval(() => {
      setCounter((p) => p + 1);
    }, 1000);
    return () => clearInterval(i);
  }, []);
  return XTimeStamp.since(stamp);
};

export const TimeSpanSince = <E extends Generic.ElementType = "p">({
  children,
  ...rest
}: TimeSpanSinceProps<E>): ReactElement => {
  const span = useTimeSpanSince(children);
  return <TimeSpan {...rest}>{span}</TimeSpan>;
};
