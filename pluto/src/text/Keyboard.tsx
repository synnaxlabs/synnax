// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type TextProps, Text } from "@/text/Text";
import { type Level } from "@/text/types";

import "@/text/Keyboard.css";

export type KeyboardProps<L extends Level = "h1"> = TextProps<L>;

export const Keyboard = <L extends Level = "p">({
  className,
  ...props
}: KeyboardProps<L>): ReactElement => (
  // @ts-expect-error
  <Text<L> className={CSS(className, CSS.BM("text", "keyboard"))} {...props} />
);
