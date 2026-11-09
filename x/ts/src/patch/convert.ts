// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { caseconv } from "@/caseconv";
import { type Action } from "@/patch/patch";
import { type Guard } from "@/patch/types.gen";
import { zod } from "@/zod";

type PathConverter = typeof caseconv.snakeToCamelPath;

const convert = (
  action: Action,
  converter: PathConverter,
  opts: caseconv.PathOptions,
): Action => {
  const at = (path: string[]) => converter(path, opts);
  const convertGuards = (gs: Guard[]): Guard[] =>
    gs.map((g) => ({ path: at(g.path).path, value: g.value }));
  switch (action.type) {
    case "set": {
      const { path, value, guards: gs } = action.set;
      const converted = at(path);
      return {
        type: "set",
        set: {
          path: converted.path,
          value: converted.convert(value),
          guards: convertGuards(gs),
        },
      };
    }
    case "remove": {
      const { path, key, guards: gs } = action.remove;
      const converted = at([...path, key]).path;
      return {
        type: "remove",
        remove: {
          path: converted.slice(0, -1),
          key: converted[converted.length - 1],
          guards: convertGuards(gs),
        },
      };
    }
    case "insert": {
      const { path, value, after, guards: gs } = action.insert;
      const converted = at(path);
      return {
        type: "insert",
        insert: {
          path: converted.path,
          value: converted.convert([value])[0],
          after,
          guards: convertGuards(gs),
        },
      };
    }
    case "move": {
      const { path, key, after, guards: gs } = action.move;
      return {
        type: "move",
        move: { path: at(path).path, key, after, guards: convertGuards(gs) },
      };
    }
  }
};

/**
 * Converts an action from the snake_case form the wire carries into the camelCase form
 * of the document, the way the codec converts the whole document with the same schema.
 */
export const snakeToCamel = (action: Action, opts: caseconv.PathOptions = {}): Action =>
  convert(action, caseconv.snakeToCamelPath, opts);

/** Converts an action from the camelCase form of the document into its wire form. */
export const camelToSnake = (action: Action, opts: caseconv.PathOptions = {}): Action =>
  convert(action, caseconv.camelToSnakePath, opts);

/**
 * Builds the guards for a path: one for each discriminated union that the path goes
 * into, holding the discriminator value that the document holds there.
 *
 * @throws {Error} if the schema does not contain the path, or a union that the path
 * goes into has a discriminator that is not a string.
 */
export const guards = (schema: z.ZodType, path: string[], doc: unknown): Guard[] =>
  zod.getDiscriminators(schema, path, doc).map(({ path, value }) => {
    if (typeof value !== "string")
      throw new Error(`The discriminator at ${path.join(".")} is not a string`);
    return { path, value };
  });
