// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/breadcrumb/Breadcrumb.css";

import { Children, Fragment, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { type Generic } from "@/generic";
import { Icon } from "@/icon";
import { Text } from "@/text";

/** Which segments of a breadcrumb are drawn in the full text color. */
export type HighlightVariant = "last" | "first" | "all";

/** Props for {@link Breadcrumb}. */
export type BreadcrumbProps<E extends Generic.ElementType = "p"> = Omit<
  Text.TextProps<E>,
  "children"
> & {
  children: ReactNode;
  highlightVariant?: HighlightVariant;
};

const Separator = () => (
  <Icon.Caret.Right className={CSS.BE("breadcrumb", "separator")} />
);

/** Props for {@link Segment}. */
export type SegmentProps<E extends Generic.ElementType = "span"> = Text.TextProps<E>;

/** One step of a {@link Breadcrumb}. A plain string child becomes one on its own. */
export const Segment = <E extends Generic.ElementType = "span">({
  children,
  ...rest
}: SegmentProps<E>): ReactElement => (
  <Text.Text className={CSS.BE("breadcrumb", "segment")} {...rest} defaultEl="span">
    {children}
  </Text.Text>
);

/**
 * A path rendered as segments with carets between them.
 *
 * @example <Breadcrumb.Breadcrumb highlightVariant="last">{parts}</Breadcrumb.Breadcrumb>
 */
export const Breadcrumb = ({
  children,
  highlightVariant,
  ...rest
}: BreadcrumbProps): ReactElement => (
  <Text.Text
    className={CSS.cls(
      CSS.B("breadcrumb"),
      highlightVariant != null && CSS.BM("breadcrumb", "highlight", highlightVariant),
    )}
    x
    align="center"
    gap="small"
    {...rest}
  >
    {Children.map(children, (child, i) => {
      if (child == null || typeof child === "boolean") return null;
      return (
        <Fragment key={i}>
          {i > 0 && <Separator />}
          {typeof child === "string" ? <Segment>{child}</Segment> : child}
        </Fragment>
      );
    })}
  </Text.Text>
);

interface MapURLSegmentsProps {
  href: string;
  segment: string;
  index: number;
}

/** Renders one element per path segment, each given the URL up to that point. */
export const mapURLSegments = (
  url: string,
  callback: (props: MapURLSegmentsProps) => ReactElement,
) => {
  const segments = url.split("/");
  return segments.map((segment, i) =>
    callback({ href: segments.slice(0, i + 1).join("/"), segment, index: i }),
  );
};
