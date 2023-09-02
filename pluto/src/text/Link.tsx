// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ForwardedRef, forwardRef, type ReactElement } from "react";

import { CSS } from "@/css";
import { Text, type TextProps } from "@/text/Text";
import { type Level } from "@/text/types";

import "@/text/Link.css";

export type LinkProps<L extends Level = "h1"> = Omit<TextProps<L>, "ref"> & {
  href?: string;
  download?: string;
  target?: string;
  rel?: string;
};

const CoreLink = <L extends Level = "h1">(
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
    {/* @ts-expect-error */}
    <Text<L> className={CSS.BE("text-link", "text")} {...props} />
  </a>
);

// @ts-expect-error
export const Link = forwardRef(CoreLink) as <L extends Level = "h1">(
  props: LinkProps<L> & { ref?: ForwardedRef<HTMLAnchorElement> },
) => ReactElement;
