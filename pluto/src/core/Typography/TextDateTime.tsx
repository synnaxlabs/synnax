// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text } from "./Text";
import type { TextProps } from "./Text";

import { timeStringFormatters, TimeStringFormatter } from "@/util/time";

export interface TextDateTimeProps extends Omit<TextProps, "children"> {
  children: number;
  format?: TimeStringFormatter;
}

export const TextDateTime = ({
  format = "shortDateTime",
  children,
  ...props
}: TextDateTimeProps): JSX.Element => {
  const formatter = timeStringFormatters[format];
  return <Text {...props}>{formatter(children)}</Text>;
};
