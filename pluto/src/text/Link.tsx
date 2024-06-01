// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Link.css";

import { type ForwardedRef, forwardRef, type ReactElement } from "react";

import { CSS } from "@/css";
import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";

export type LinkProps<L extends text.Level = "h1"> = Omit<TextProps<L>, "ref"> & {
  href?: string;
  download?: string;
  target?: string;
  rel?: string;
};

const CoreLink = <L extends text.Level = "h1">(
  { href, download, target, rel, ...props }: LinkProps<L>,
  ref: ForwardedRef<HTMLAnchorElement>,
): ReactElement => (
  <a
    className={CSS.B("text-link")}
    ref={ref}
    href={href}
    download={download}
    target={target}
    rel={rel}
  >
    {/* @ts-expect-error - generic component errors */}
    <Text<L> className={CSS.BE("text-link", "text")} {...props} />
  </a>
);

// @ts-expect-error - generic component errors
export const Link = forwardRef(CoreLink) as <L extends text.Level = "h1">(
  props: LinkProps<L> & { ref?: ForwardedRef<HTMLAnchorElement> },
) => ReactElement;
