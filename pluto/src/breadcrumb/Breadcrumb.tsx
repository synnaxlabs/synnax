// Copyright 2025 Synnax Labs, Inc.
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

export type BreadcrumbProps<E extends Generic.ElementType = "p"> = Omit<
  Text.TextProps<E>,
  "children"
> & {
  children: ReactNode;
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

/**
 * Breadcrumb component for displaying a breadcrumb navigation.
 *
 * @template E - The type of the space element.
 * @template L - The text level.
 *
 * @param props - The props for the Breadcrumb component.
 * @returns The Breadcrumb component.
 */
export const Breadcrumb = ({ children, ...rest }: BreadcrumbProps): ReactElement => (
  <Text.Text
    className={CSS(CSS.B("breadcrumb"))}
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
