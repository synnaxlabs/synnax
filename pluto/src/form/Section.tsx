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
import { Text } from "@/text";

export interface SectionsProps extends Flex.BoxProps {}

/**
 * Lays out sections. Stacked (the default), every label in every section shares one
 * column sized to the widest label and every control fills the rest. Side by side
 * (`x`), each section is as wide as its content: fields fill the rows the height
 * affords, then continue in a further label and control column pair.
 */
export const Sections = ({
  className,
  direction,
  x,
  y,
  ...rest
}: SectionsProps): ReactElement => {
  const dir = Flex.parseDirection(direction, x, y) ?? "y";
  return (
    <Flex.Box
      className={CSS.cls(CSS.B("form-sections"), CSS.dir(dir), className)}
      empty
      {...rest}
    />
  );
};

export interface SectionProps extends Omit<Flex.BoxProps, "title"> {
  title: string;
  actions?: ReactNode;
}

/** A titled group of fields inside `Sections`. */
export const Section = ({
  title,
  actions,
  children,
  className,
  ...rest
}: SectionProps): ReactElement => (
  <Flex.Box className={CSS.cls(CSS.B("form-section"), className)} empty {...rest}>
    <Flex.Box x align="center" gap="small" className={CSS.BE("form-section", "header")}>
      <Text.Text level="p" weight={500} color={10}>
        {title}
      </Text.Text>
      {actions}
    </Flex.Box>
    <div className={CSS.BE("form-section", "body")}>{children}</div>
  </Flex.Box>
);
