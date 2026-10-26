// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/form/Section.css";

import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useCombinedRefs, useWheelScrollX } from "@/hooks";
import { Text } from "@/text";

export interface SectionsProps extends Flex.BoxProps {}

/**
 * Lays out sections. Stacked (the default), every label in every section shares one
 * column sized to the widest label and every control fills the rest. Side by side
 * (`x`), each section is as wide as its content: fields fill three rows, then
 * continue in a further label and control column pair. The side-by-side strip
 * scrolls horizontally under a vertical mouse wheel.
 */
export const Sections = ({
  className,
  direction,
  x,
  y,
  ref,
  ...rest
}: SectionsProps): ReactElement => {
  const dir = Flex.parseDirection(direction, x, y) ?? "y";
  const wheelRef = useWheelScrollX<HTMLDivElement>();
  const combinedRef = useCombinedRefs(ref, wheelRef);
  return (
    <Flex.Box
      className={CSS.cls(CSS.B("form-sections"), className)}
      direction={dir}
      ref={dir === "x" ? combinedRef : ref}
      empty
      {...rest}
    />
  );
};

export interface SectionProps extends Omit<Flex.BoxProps, "title"> {
  /** Omit it when the section's container already names it, such as a tab. */
  title?: string;
  /** Shown in the header, so they need a title. */
  actions?: ReactNode;
}

/** A group of fields inside `Sections`, under a header when it has a title. */
export const Section = ({
  title,
  actions,
  children,
  className,
  ...rest
}: SectionProps): ReactElement => (
  <Flex.Box className={CSS.cls(CSS.B("form-section"), className)} empty {...rest}>
    {title != null && (
      <Flex.Box
        x
        align="center"
        gap="small"
        className={CSS.BE("form-section", "header")}
      >
        <Text.Text level="h5" weight={500} color={11}>
          {title}
        </Text.Text>
        {actions}
      </Flex.Box>
    )}
    <div className={CSS.BE("form-section", "body")}>{children}</div>
  </Flex.Box>
);
