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
import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";

export type LinkProps<L extends text.Level = "h1"> = TextProps<L> & {
  href?: string;
  download?: string;
  target?: string;
  rel?: string;
};

export const Link = <L extends text.Level = "h1">({
  ref,
  href,
  download,
  target,
  rel,
  className,
  ...rest
}: LinkProps<L>): ReactElement => (
  // @ts-expect-error - generic component errors
  <Text<L>
    el="a"
    ref={ref}
    href={href}
    download={download}
    target={target}
    rel={rel}
    className={CSS(className, CSS.B("text-link"))}
    {...rest}
  />
);
