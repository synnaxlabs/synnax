// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CrudeTimeStamp,
  TimeStamp,
  type TimeStampStringFormat,
  type TZInfo,
} from "@synnaxlabs/x";
import { type ForwardedRef, forwardRef, type JSX, type ReactElement } from "react";

import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";

export type DateTimeProps<L extends text.Level = "h1"> = Omit<
  TextProps<L>,
  "children" | "ref"
> & {
  children: CrudeTimeStamp;
  format?: TimeStampStringFormat;
  suppliedTZ?: TZInfo;
  displayTZ?: TZInfo;
};

export const CoreDateTime = <L extends text.Level = "h1">(
  {
    format = "dateTime",
    suppliedTZ = "UTC",
    displayTZ = "local",
    children,
    ...props
  }: DateTimeProps<L>,
  ref: ForwardedRef<JSX.IntrinsicElements[L]>,
): ReactElement => (
  // @ts-expect-error - generic component errors
  <Text<L> ref={ref} {...props}>
    {new TimeStamp(children, suppliedTZ).fString(format, displayTZ)}
  </Text>
);

// @ts-expect-error - generic component errors
export const DateTime = forwardRef(CoreDateTime) as <L extends text.Level = "h1">(
  props: DateTimeProps<L> & { ref?: React.Ref<L> },
) => ReactElement;
