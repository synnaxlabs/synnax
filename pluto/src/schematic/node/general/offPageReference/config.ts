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

/** An ontology ID narrowed to the page types a reference can target. */
export const pageZ = z.object({ type: pageTypeZ, key: z.string() });
export interface Page extends z.infer<typeof pageZ> {}

/**
 * Normalizes a stored page config value. A bare string is a legacy schematic key. An
 * unrecognized value normalizes to an empty schematic key, meaning no page.
 */
export const parsePage = (page?: unknown): Page => {
  if (typeof page === "string") return { type: "schematic", key: page };
  const parsed = pageZ.safeParse(page);
  return parsed.success ? parsed.data : { type: "schematic", key: "" };
};

export const configZ = z.object({
  variant: z.literal(VARIANT),
  orientation: location.outerZ.optional(),
  label: Label.configZ,
  level: text.levelZ.optional(),
  color: color.crudeZ.optional(),
  page: pageZ.or(z.string()).optional(),
  dblClickNav: z.boolean().optional(),
});
export type Config = z.infer<typeof configZ>;
