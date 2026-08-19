// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Text.css";

import { type status } from "@synnaxlabs/client";
import { type text } from "@synnaxlabs/x";
import {
  Children,
  type ComponentPropsWithoutRef,
  type ReactElement,
  type ReactNode,
  useMemo,
} from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Generic } from "@/generic";
import { isValidElement } from "@/util/children";

type AnchorProps = ComponentPropsWithoutRef<"a">;

/** The typeface and treatment: body prose, monospace code, a key cap, or a link. */
export type Variant = "prose" | "code" | "keyboard" | "link";
/** What happens to text that outruns its box. */
export type Overflow = "ellipsis" | "fade" | "clip" | "nowrap" | "wrap";

/**
 * The text-specific props {@link TextProps} adds. Text is a flex box, so it also takes
 * every {@link Flex.BoxExtensionProps} for laying out icons beside a label.
 */
export interface ExtensionProps
  extends Flex.BoxExtensionProps, Pick<AnchorProps, "href" | "target" | "rel"> {
  /** The type scale step, which also picks the rendered element: p, h1, h2. */
  level?: text.Level;
  children?: ReactNode;
  /** The font weight. Defaults to the level's own. */
  weight?: text.Weight;
  /** The typeface treatment. Defaults to "prose". */
  variant?: Variant;
  /** Prefixes a scheme-less href with `https://`. */
  autoFormatHref?: boolean;
  /** The element to render when `level` does not imply one. */
  defaultEl?: Generic.ElementType;
  /** What to do with text that outruns its box. */
  overflow?: Overflow;
  /** Tints the text with a status color. */
  status?: status.Variant;
  /** Truncates past this many lines. Implies an ellipsis. */
  lineClamp?: number;
}

/** Props for {@link Text}, on top of the props of the element it renders as. */
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

/**
 * @returns true if the children are a lone icon, which the chassis renders in a square
 * rather than a padded pill.
 */
export const isSquare = (children: ReactNode): boolean => {
  if (Children.count(children) !== 1) {
    const parsedChildren = Children.toArray(children).filter(
      (c) => typeof c !== "boolean" && c != " ",
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

/** Resolves which element a text-based component renders as. */
export const parseElement = <E extends Generic.ElementType = "p">(
  level?: text.Level,
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

/**
 * Renders text at a step on the shared type scale. It is also a flex box, so icons and
 * a label lay out inside one element with no wrapper.
 *
 * @example <Text.Text level="h3">Ranges</Text.Text>
 * @example <Text.Text level="p" overflow="ellipsis"><Icon.Range />{name}</Text.Text>
 */
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
}: TextProps<E>): ReactElement => {
  const formattedStyle = useMemo(
    () => formatStyle(style, weight, lineClamp),
    [style, weight, lineClamp],
  );
  return (
    <Flex.Box<E>
      direction="x"
      el={parseElement<E>(level, el, defaultEl, variant, href)}
      style={formattedStyle}
      className={CSS.cls(
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
};
