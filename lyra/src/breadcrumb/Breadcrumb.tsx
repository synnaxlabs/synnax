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

export type HighlightVariant = "last" | "first" | "all";

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

export type SegmentProps<E extends Generic.ElementType = "span"> = Text.TextProps<E>;

export const Segment = <E extends Generic.ElementType = "span">({
  children,
  ...rest
}: SegmentProps<E>): ReactElement => (
  <Text.Text className={CSS.BE("breadcrumb", "segment")} {...rest} defaultEl="span">
    {children}
  </Text.Text>
);

export const Breadcrumb = ({
  children,
  highlightVariant,
  ...rest
}: BreadcrumbProps): ReactElement => (
  <Text.Text
    className={CSS(
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

export const mapURLSegments = (
  url: string,
  callback: (props: MapURLSegmentsProps) => ReactElement,
) => {
  const segments = url.split("/");
  return segments.map((segment, i) =>
    callback({ href: segments.slice(0, i + 1).join("/"), segment, index: i }),
  );
};
