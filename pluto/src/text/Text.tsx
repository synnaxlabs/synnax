// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Text.css";

import { type status } from "@synnaxlabs/x";
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
export type Overflow = "ellipsis" | "clip" | "nowrap" | "wrap";

export interface ExtensionProps
  extends Flex.BoxExtensionProps, Pick<AnchorProps, "href" | "target" | "rel"> {
  /* The level of text to display i.e. p, h1, h2 */
  level?: text.Level;
  /* The text to display */
  children?: ReactNode;
  /* Shade sets the shade of the text */
  /* Weight sets the weight of the text */
  weight?: text.Weight;
  /* Variant sets the variant of the text */
  variant?: Variant;
  /* AutoFormatHref formats the href to be a secure http link */
  autoFormatHref?: boolean;
  /* DefaultEl sets the default element of the text */
  defaultEl?: Generic.ElementType;
  /* Overflow sets the overflow of the text */
  overflow?: Overflow;
  /* Status sets the status of the text */
  status?: status.Variant;
  /* LineClamp sets the maximum number of lines before truncating with ellipsis */
  lineClamp?: number;
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

export const isSquare = (children: ReactNode): boolean => {
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

const parseElement = <E extends Generic.ElementType = "p">(
  level: text.Level,
  el?: E,
  defaultEl?: Generic.ElementType,
  variant?: Variant,
  href?: string,
): E | undefined => {
  if (el != null) return el;
  if (href != null || variant === "link") return "a" as E;
  if (defaultEl != null) return defaultEl as E;
  if (level != null) return level as E;
  return "p" as E;
};

const formatStyle = (
  base: React.CSSProperties | undefined,
  weight: text.Weight | undefined,
  lineClamp: number | undefined,
): React.CSSProperties => {
  const style: React.CSSProperties = {
    ...base,
    fontWeight: weight,
  };
  if (lineClamp != null) style.WebkitLineClamp = lineClamp;
  return style;
};

export const Text = <E extends Generic.ElementType = "p">({
  level = "p",
  className,
  style,
  weight,
  defaultEl,
  el,
  variant,
  overflow,
  href,
  autoFormatHref,
  status,
  lineClamp,
  ...rest
}: TextProps<E>): ReactElement => (
  <Flex.Box<E>
    direction="x"
    el={parseElement<E>(level, el, defaultEl, variant, href)}
    style={formatStyle(style, weight, lineClamp)}
    className={CSS(
      CSS.B("text"),
      variant != null && CSS.BM("text", variant),
      CSS.BM("text", level),
      overflow != null && CSS.BM("text", "overflow", overflow),
      lineClamp != null && CSS.BM("text", "line-clamp"),
      status != null && CSS.M("status", status),
      className,
    )}
    square={isSquare(rest.children)}
    gap="small"
    href={formatHref(href, autoFormatHref)}
    {...(rest as Flex.BoxProps<E>)}
  />
);
