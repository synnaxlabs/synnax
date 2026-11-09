// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { deep } from "@/deep";
import { narrow } from "@/narrow";
import { guards } from "@/patch/convert";
import { type Action } from "@/patch/patch";

/**
 * Lists the actions that turn base into next and touch only what differs, so a
 * concurrent edit to another field survives. Objects compare field by field, and a
 * field that next does not hold is removed. Any other value that differs is set whole,
 * as is an object whose discriminated union changes member.
 *
 * @param schema - The schema of base and next, which decides the guards of each action.
 * @throws {Error} if the schema does not contain a path that differs.
 */
export const diff = (schema: z.ZodType, base: unknown, next: unknown): Action[] => {
  const actions: Action[] = [];
  const guardsAt = (path: string[]) => guards(schema, path, base);
  const isDiscriminator = (path: string[]): boolean =>
    guardsAt(path).some((guard) => deep.equal(guard.path, path));
  const set = (path: string[], value: unknown): void => {
    actions.push({ type: "set", set: { path, value, guards: guardsAt(path) } });
  };
  const walk = (path: string[], prev: unknown, cur: unknown): void => {
    if (deep.equal(prev, cur)) return;
    if (!narrow.isPlainObject(prev) || !narrow.isPlainObject(cur)) {
      set(path, cur);
      return;
    }
    const changesMember = Object.keys(prev).some(
      (key) => prev[key] !== cur[key] && isDiscriminator([...path, key]),
    );
    if (changesMember) {
      set(path, cur);
      return;
    }
    for (const key of new Set([...Object.keys(prev), ...Object.keys(cur)])) {
      const child = [...path, key];
      if (cur[key] !== undefined) walk(child, prev[key], cur[key]);
      else if (prev[key] !== undefined)
        actions.push({
          type: "remove",
          remove: { path, key, guards: guardsAt(child) },
        });
    }
  };
  walk([], base, next);
  return actions;
};
