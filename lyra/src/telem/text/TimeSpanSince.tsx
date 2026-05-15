// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@synnaxlabs/x/telem";
import { type ReactElement, useEffect, useState } from "react";

import { type Generic } from "@/generic";
import { TimeSpan, type TimeSpanProps } from "@/telem/text/TimeSpan";

export type TimeSpanSinceProps<E extends Generic.ElementType = "p"> = Omit<
  TimeSpanProps<E>,
  "children"
> & {
  children: telem.CrudeTimeStamp;
  format?: telem.TimeSpanStringFormat;
};

export const useTimeSpanSince = (stamp: telem.CrudeTimeStamp): telem.TimeSpan => {
  const [, setCounter] = useState(0);
  useEffect(() => {
    const i = setInterval(() => {
      setCounter((p) => p + 1);
    }, 1000);
    return () => clearInterval(i);
  }, []);
  return telem.TimeStamp.since(stamp);
};

export const TimeSpanSince = <E extends Generic.ElementType = "p">({
  children,
  ...rest
}: TimeSpanSinceProps<E>): ReactElement => {
  const span = useTimeSpanSince(children);
  return <TimeSpan {...rest}>{span}</TimeSpan>;
};
