// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { caseconv, type Optional, toArray } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Icon as PIcon } from "@/icon";
import { Text } from "@/text";
import { isValidElement } from "@/util/children";

export interface Segment {
  label: string;
  shade?: Text.Shade;
  icon?: string | ReactElement;
  weight?: Text.Weight;
  level?: Text.Level;
}

export type Segments = string | Segment | (string | Segment)[];

/**
 * Props for the Breadcrumb component.
 *
 * @template E - The type of the space element.
 * @template L - The text level.
 */
export type BreadcrumbProps<
  E extends Align.SpaceElementType = "div",
  L extends Text.Level = Text.Level,
> = Optional<Omit<Text.WithIconProps<E, L>, "children">, "level"> & {
  /** Icon to display in the breadcrumb. */
  icon?: string | ReactElement;
  /** The breadcrumb items, either a single string or an array of strings. */
  children: Segments;
  url?: string | string[];
  /** Separator to use between breadcrumb items. Defaults to ".". */
  separator?: string;
  /** Whether to hide the first breadcrumb item. */
  hideFirst?: boolean;
};

const normalizeSegments = (
  segments: string | Segment | (string | Segment)[],
  separator: string,
): Segment[] => {
  const arr = toArray(segments)
    .map((segment) => {
      if (typeof segment === "string")
        return segment.split(separator).map((label) => ({ label }));
      return segment;
    })
    .flat();
  return arr;
};

interface GetContentArgs {
  segments: Segments;
  shade: Text.Shade;
  level: Text.Level;
  weight?: Text.Weight;
  separator: string;
}

const getContent = ({
  segments,
  shade,
  level,
  weight,
  separator,
}: GetContentArgs): (ReactElement | string)[] => {
  const normalized = normalizeSegments(segments, separator);
  const content: (ReactElement | string)[] = normalized
    .map((el, index) => {
      const base: ReactElement[] = [
        <Icon.Caret.Right
          key={`${el}-${index}`}
          style={{
            transform: "scale(0.8) translateY(1px)",
            color: CSS.shade(shade),
          }}
        />,
      ];
      if (el.icon != null) {
        const icon = PIcon.resolve(el.icon);
        if (icon != null) base.push(icon);
      }
      base.push(
        <Text.Text
          style={{ marginLeft: el.icon != null ? "0.5rem" : "0" }}
          key={el.label}
          weight={weight}
          shade={shade}
          level={level}
          {...el}
        >
          {el.label}
        </Text.Text>,
      );
      return base;
    })

    .flat();
  return content;
};

/**
 * Breadcrumb component for displaying a breadcrumb navigation.
 *
 * @template E - The type of the space element.
 * @template L - The text level.
 *
 * @param props - The props for the Breadcrumb component.
 * @returns The Breadcrumb component.
 */
export const Breadcrumb = <
  E extends Align.SpaceElementType = "div",
  L extends Text.Level = Text.Level,
>({
  children,
  icon,
  shade = 7,
  weight,
  size = 0.5,
  url,
  level = "p",
  separator = ".",
  className,
  hideFirst = true,
  ...props
}: BreadcrumbProps<E, L>): ReactElement => {
  if (url != null) children = url;
  const content = getContent({ segments: children, separator, shade, level, weight });
  if (hideFirst) content.shift();
  return (
    <Text.WithIcon
      className={CSS(className, CSS.B("breadcrumb"))}
      level={level}
      shade={shade}
      weight={weight}
      size={size}
      direction="x"
      {...props}
    >
      {PIcon.resolve(icon)}
      {...content}
    </Text.WithIcon>
  );
};

export interface URLProps extends Omit<BreadcrumbProps, "children" | "url"> {
  url: string;
}

export const URL = ({
  url,
  className,
  level = "p",
  separator = ".",
  weight,
  shade = 7,
}: URLProps) => {
  const content = getContent({ segments: url, separator, shade, level, weight });
  return (
    <Align.Space
      className={CSS(className, CSS.B("breadcrumb"))}
      direction="x"
      size="small"
      align="center"
    >
      {content.map((el, index) => {
        if (isValidElement(el)) return el;
        if (el == null) return null;
        const split = url.split(separator);
        const idx = split.indexOf(el);
        let href = url
          .split(separator)
          .slice(0, idx + 1)
          .join("/");
        href = `/${href}`;
        return (
          <Text.Link level={level} key={index} href={href}>
            {el
              .split("-")
              .map((el) => caseconv.capitalize(el))
              .join(" ")}
          </Text.Link>
        );
      })}
    </Align.Space>
  );
};
