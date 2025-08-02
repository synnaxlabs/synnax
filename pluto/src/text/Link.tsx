// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Link.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type Generic } from "@/generic";
import { Text, type TextProps } from "@/text/Text";

export type LinkProps<E extends Generic.ElementType = "a"> = TextProps<E> & {
  href?: string;
  download?: string;
  target?: string;
  rel?: string;
};

export const Link = <E extends Generic.ElementType = "a">({
  className,
  ...rest
}: LinkProps<E>): ReactElement => (
  <Text<E>
    el="a"
    className={CSS(className, CSS.B("text-link"))}
    {...(rest as TextProps<E>)}
  />
);
