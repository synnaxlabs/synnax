// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, patch, type record, zod } from "@synnaxlabs/x";
import { type Draft } from "immer";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/ranger/actions.gen";
import { type Payload, payloadZ } from "@/ranger/types.gen";

/** The part of a range that path actions edit. */
const documentZ = payloadZ.pick({ name: true, timeRange: true, color: true }).strict();

// Actions hold the wire form of a range: snake_case keys and nanosecond timestamps.
// Immer cannot draft the cached range's TimeRange, so each action applies to the wire
// form of the range, which then decodes back.
const applyPath = (state: Draft<Payload>, action: patch.Action): HandlerResult => {
  const { name, timeRange, color } = state;
  const doc: record.Unknown = caseconv.camelToSnake(
    { name, timeRange, color },
    { schema: documentZ },
  );
  const inverse = patch.apply(doc, action);
  if (inverse.length === 0) return actions.NO_OP_RESULT;
  const next = zod.parse(documentZ, caseconv.snakeToCamel(doc, { schema: documentZ }), {
    label: "range",
  });
  state.name = next.name;
  state.timeRange = next.timeRange;
  if (next.color == null) delete state.color;
  else state.color = next.color;
  return { inverse, targets: [patch.pathOf(action).join(".")] };
};

const handlers: Handlers = {
  create: (state, { range }) => {
    for (const key of Object.keys(state) as Array<keyof Payload>)
      if (!(key in range)) delete state[key];
    Object.assign(state, range);
    return { inverse: [], targets: [range.key] };
  },
  set: (state, set) => applyPath(state, { type: "set", set }),
  remove: (state, remove) => applyPath(state, { type: "remove", remove }),
  insert: (state, insert) => applyPath(state, { type: "insert", insert }),
  move: (state, move) => applyPath(state, { type: "move", move }),
};

/**
 * Applies actions to a range in order. Path actions edit only the range's name, time
 * range, and color.
 * @throws {Error} if the range cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the range that a create action carries, or undefined for other actions. */
export const createOf = (action: Action): Payload | undefined =>
  action.type === "create" ? action.create.range : undefined;
