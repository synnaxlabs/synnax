// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@synnaxlabs/x/telem";
import { type ReactElement } from "react";

import { type Generic } from "@/generic";
import { Text } from "@/text";
export type TimeStampProps<E extends Generic.ElementType = "p"> = Omit<
  Text.TextProps<E>,
  "children"
> & {
  children: telem.CrudeTimeStamp;
  format?: telem.TimeStampStringFormat;
  suppliedTZ?: telem.TZInfo;
  displayTZ?: telem.TZInfo;
};

export const TimeStamp = <E extends Generic.ElementType = "p">({
  format = "dateTime",
  suppliedTZ = "UTC",
  displayTZ = "local",
  children,
  ...rest
}: TimeStampProps<E>): ReactElement => (
  <Text.Text<E> {...(rest as Text.TextProps<E>)}>
    {new telem.TimeStamp(children, suppliedTZ).toString(format, displayTZ)}
  </Text.Text>
);
