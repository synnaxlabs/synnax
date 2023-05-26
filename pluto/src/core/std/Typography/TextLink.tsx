// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ForwardedRef, forwardRef, ReactElement } from "react";

import { CSS } from "@/core/css";
import { Text, TextProps } from "@/core/std/Typography/Text";
import { TypographyLevel } from "@/core/std/Typography/types";

import "@/core/Typography/TextLink.css";

export type TextLinkProps<L extends TypographyLevel = "h1"> = Omit<
  TextProps<L>,
  "ref"
> & {
  href?: string;
  download?: string;
  target?: string;
  rel?: string;
};

const CoreTextLink = <L extends TypographyLevel = "h1">(
  { href, download, target, rel, ...props }: TextLinkProps<L>,
  ref: ForwardedRef<HTMLAnchorElement>
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
export const TextLink = forwardRef(CoreTextLink) as <L extends TypographyLevel = "h1">(
  props: TextLinkProps<L> & { ref?: ForwardedRef<HTMLAnchorElement> }
) => ReactElement;
