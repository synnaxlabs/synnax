// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/section/Section.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** Props for {@link Section}. */
export interface SectionProps extends Omit<
  Flex.BoxProps,
  "bordered" | "borderColor" | "rounded" | "background"
> {}

/**
 * A section of a page, such as the channels of a task form: an outlined box with large
 * corners on the l1 surface. It clips its content to its corners.
 */
export const Section = ({ className, ...rest }: SectionProps): ReactElement => (
  <Flex.Box
    bordered
    borderColor={6}
    rounded="large"
    background={1}
    className={CSS.cls(CSS.B("section"), className)}
    {...rest}
  />
);
