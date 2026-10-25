// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/skeleton/Skeleton.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

/**
 * A pulsing block that holds the place of content still loading. It is one line of the
 * surrounding text tall, so it can sit in a text element in place of the text. Size it
 * across with the box props or a style.
 */
export const Skeleton = ({
  className,
  ...rest
}: Omit<Flex.BoxProps<"span">, "children">): ReactElement => (
  <Flex.Box el="span" className={CSS.cls(CSS.B("skeleton"), className)} {...rest} />
);
