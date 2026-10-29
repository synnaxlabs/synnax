// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, color, patch, type record, TimeRange, zod } from "@synnaxlabs/x";
import { type Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/ranger/actions.gen";
import { type Payload } from "@/ranger/types.gen";

// The part of a range that path actions edit. It checks shape but not validity: the
// Core validates, and a form holds an invalid value unsent while the cache shows it.
const documentZ = z
  .object({ name: z.string(), timeRange: TimeRange.z, color: color.colorZ.optional() })
  .strict();

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
  const path = patch.pathOf(action);
  // Only the field the action reaches changes, so the others keep their identity.
  if (path[0] === "name") state.name = next.name;
  else if (path[0] === "time_range") state.timeRange = next.timeRange;
  else if (next.color == null) delete state.color;
  else state.color = next.color;
  return { inverse, targets: [path.join(".")] };
};

const handlers: Handlers = {
  create: (state, { range }) => {
    for (const key of Object.keys(state) as Array<keyof Payload>)
      if (!(key in range)) delete state[key];
    Object.assign(state, range);
    return { inverse: [], targets: [range.key] };
  },
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a range in order. Path actions edit only the range's name, time
 * range, and color.
 * @throws {Error} if the range cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/**
 * Converts an action from the camelCase form of the cached range into the snake_case
 * form that the Core stores.
 */
export const encode = (state: Payload, action: Action): Action =>
  action.type === "create"
    ? action
    : patch.camelToSnake(action, { schema: documentZ, data: state });

/** @returns the range that a create action carries, or undefined for other actions. */
export const createOf = (action: Action): Payload | undefined =>
  action.type === "create" ? action.create.range : undefined;
