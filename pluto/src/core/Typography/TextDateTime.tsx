// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import { Text } from "./Text";
import type { TextProps } from "./Text";

import {
  TimeStampStringFormat,
  TimeStamp,
  UnparsedTimeStamp,
  TZInfo,
} from "@synnaxlabs/x";

export interface TextDateTimeProps extends Omit<TextProps, "children" | "ref"> {
  children: UnparsedTimeStamp;
  format?: TimeStampStringFormat;
  suppliedTZ?: TZInfo;
  displayTZ?: TZInfo;
}

export const TextDateTime = forwardRef<HTMLParagraphElement, TextDateTimeProps>(
  (
    {
      format = "dateTime",
      suppliedTZ = "UTC",
      displayTZ = "local",
      children,
      ...props
    },
    ref
  ): JSX.Element => (
    <Text ref={ref} {...props}>
      {new TimeStamp(children, suppliedTZ).fString(format, displayTZ)}
    </Text>
  )
);
TextDateTime.displayName = "TextDateTime";
