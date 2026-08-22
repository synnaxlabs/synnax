// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/form/Section.css";

import { Flex, Text } from "@synnaxlabs/pluto";
import { type ComponentPropsWithoutRef, type ReactNode } from "react";

import { CSS } from "@/platform/css";

export type SectionsProps = ComponentPropsWithoutRef<"div">;

/**
 * Lays out its sections on one grid, so every label shares a column sized to the
 * widest label and every control fills the rest.
 */
export const Sections = ({ className, ...rest }: SectionsProps) => (
  <div className={CSS.cls(CSS.B("form-sections"), className)} {...rest} />
);

export interface SectionProps extends Omit<ComponentPropsWithoutRef<"div">, "title"> {
  title: string;
  actions?: ReactNode;
}

/** A titled group of fields inside a `Sections` grid. */
export const Section = ({
  title,
  actions,
  children,
  className,
  ...rest
}: SectionProps) => (
  <div className={CSS.cls(CSS.B("form-section"), className)} {...rest}>
    <Flex.Box x align="center" gap="small" className={CSS.BE("form-section", "header")}>
      <Text.Text level="p" weight={500} color={10}>
        {title}
      </Text.Text>
      {actions}
    </Flex.Box>
    {children}
  </div>
);
