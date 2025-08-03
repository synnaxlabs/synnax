// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Text.css";

import {
  Children,
  type ComponentPropsWithoutRef,
  type ReactElement,
  type ReactNode,
} from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Generic } from "@/generic";
import { type text } from "@/text/core";
import { isValidElement } from "@/util/children";

type AnchorProps = ComponentPropsWithoutRef<"a">;

export type Variant = "prose" | "code" | "keyboard" | "link";

export interface ExtensionProps
  extends Flex.BoxExtensionProps,
    Pick<AnchorProps, "href" | "target" | "rel"> {
  /* The level of text to display i.e. p, h1, h2 */
  level?: text.Level;
  /* The text to display */
  children?: ReactNode;
  /* NoWrap prevents the text from wrapping */
  noWrap?: boolean;
  /* Shade sets the shade of the text */
  /* Weight sets the weight of the text */
  weight?: text.Weight;
  /* Ellipsis sets whether to truncate the text */
  ellipsis?: boolean;
  /* Variant sets the variant of the text */
  variant?: Variant;
  /* AutoFormatHref formats the href to be a secure http link */
  autoFormatHref?: boolean;
  /* DefaultEl sets the default element of the text */
  defaultEl?: Generic.ElementType;
}

export type TextProps<E extends Generic.ElementType = "p"> = Omit<
  Generic.OptionalElementProps<E>,
  "color"
> &
  ExtensionProps;

const SCHEME_SEPARATOR = "://";
const HTTP_SECURE_SCHEME = `https${SCHEME_SEPARATOR}`;

const formatHref = (
  href?: string,
  autoFormatHref: boolean = false,
): string | undefined => {
  if (href == null) return href;
  if (autoFormatHref && !href.includes(SCHEME_SEPARATOR))
    return HTTP_SECURE_SCHEME + href;
  return href;
};

const isIconOnly = (children: ReactNode): boolean => {
  if (Children.count(children) !== 1) {
    const parsedChildren = Children.toArray(children).filter(
      (c) => typeof c !== "boolean",
    );
    if (parsedChildren.length !== 1) return false;
    children = parsedChildren[0];
  }
  if (typeof children === "string") return children.length === 1;
  if (
    isValidElement(children) &&
    typeof children.props === "object" &&
    children.props != null &&
    !("children" in children.props) &&
    !("role" in children.props)
  )
    return true;

  return false;
};

export const Text = <E extends Generic.ElementType = "p">({
  level = "p",
  className,
  style,
  noWrap = false,
  weight,
  defaultEl,
  el,
  variant = "prose",
  ellipsis = false,
  href,
  autoFormatHref,
  ...rest
}: TextProps<E>): ReactElement => (
  <Flex.Box<E>
    direction="x"
    el={(el ?? (href != null || variant === "link" ? "a" : (defaultEl ?? level))) as E}
    style={{ fontWeight: weight, ...style }}
    className={CSS(
      CSS.B("text"),
      CSS.BM("text", variant),
      CSS.BM("text", level),
      isIconOnly(rest.children) && CSS.BM("text", "square"),
      CSS.noWrap(noWrap),
      ellipsis && CSS.M("ellipsis"),
      className,
    )}
    gap="small"
    href={formatHref(href, autoFormatHref)}
    {...(rest as Flex.BoxProps<E>)}
  />
);
