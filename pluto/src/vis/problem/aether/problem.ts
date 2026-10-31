// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { z } from "zod";

const variantZ = z.enum(["warning", "error"]);
export type Variant = z.infer<typeof variantZ>;

/** A problem with the data that a visualization shows. */
export const problemZ = z.object({
  variant: variantZ,
  /** What the problem is about, such as the label of a line. Empty if nothing named. */
  name: z.string(),
  message: z.string(),
  description: z.string(),
});

export interface Problem extends z.infer<typeof problemZ> {}

/**
 * Converts the status of a data source into its problems.
 * @param name - What the problem is about. Defaults to the name of the status.
 * @returns the problem of the status as a list, which is empty unless the status is a
 * warning or an error.
 */
export const fromStatus = (s: status.Status, name: string = s.name): Problem[] => {
  const { variant, message, description } = s;
  if (variant !== "warning" && variant !== "error") return [];
  return [{ variant, name, message, description }];
};

/**
 * @returns the problems with the names of equal problems joined, so one cause on many
 * sources shows once.
 */
export const merge = (problems: Problem[]): Problem[] => {
  const merged = new Map<string, Problem>();
  for (const p of problems) {
    const key = JSON.stringify([p.variant, p.message, p.description]);
    const prev = merged.get(key);
    if (prev == null) merged.set(key, { ...p });
    else if (p.name !== "")
      prev.name = prev.name === "" ? p.name : `${prev.name}, ${p.name}`;
  }
  return [...merged.values()];
};
