// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, location, text } from "@synnaxlabs/x";
import { z } from "zod";

import { Icon } from "@/icon";
import { Label } from "@/schematic/node/common/label";

export const VARIANT = "offPageReference" as const;

export const PAGE_TYPES = ["schematic", "lineplot", "log", "table"] as const;
export const pageTypeZ = z.enum(PAGE_TYPES);
export type PageType = z.infer<typeof pageTypeZ>;

export const PAGE_ICONS: Record<PageType, Icon.FC> = {
  schematic: Icon.Schematic,
  lineplot: Icon.LinePlot,
  log: Icon.Log,
  table: Icon.Table,
};

export interface Page {
  type: PageType;
  key: string;
}

/**
 * Parses a page config string into its type and key. A bare string without a type
 * prefix is a legacy schematic key.
 */
export const parsePage = (page: string = ""): Page => {
  const sep = page.indexOf(":");
  if (sep !== -1) {
    const type = pageTypeZ.safeParse(page.slice(0, sep));
    if (type.success) return { type: type.data, key: page.slice(sep + 1) };
  }
  return { type: "schematic", key: page };
};

/**
 * Formats a page into its canonical config string. An empty key formats to an empty
 * string, meaning no page.
 */
export const formatPage = ({ type, key }: Page): string =>
  key.length === 0 ? "" : `${type}:${key}`;

export const configZ = z.object({
  variant: z.literal(VARIANT),
  orientation: location.outerZ.optional(),
  label: Label.configZ,
  level: text.levelZ.optional(),
  color: color.crudeZ.optional(),
  page: z.string().optional(),
  dblClickNav: z.boolean().optional(),
});
export type Config = z.infer<typeof configZ>;
