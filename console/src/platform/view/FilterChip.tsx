// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/view/FilterChip.css";

import { type Generic, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export type FilterChipProps<E extends Generic.ElementType = "p"> = Text.TextProps<E>;

/** The lead chip naming an active filter in a chip pack. */
export const FilterChip = <E extends Generic.ElementType = "p">({
  className,
  ...rest
}: FilterChipProps<E>): ReactElement => (
  <Text.Text<E>
    bordered
    size="small"
    borderColor={6}
    level="small"
    className={CSS(CSS.BE("view", "filter-chip"), className)}
    {...(rest as Text.TextProps<E>)}
  />
);
