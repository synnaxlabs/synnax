// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch } from "@synnaxlabs/x";
import { type Draft } from "immer";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/task/actions.gen";
import { type Payload, payloadZ } from "@/task/types.gen";

// The cache decodes every task with the generic schema, so actions convert with it.
const PAYLOAD_Z = payloadZ();

// Actions hold snake_case paths and values, the form the Core applies. The cached
// task holds the camelCase form that the codec decodes.
const applyPath = (state: Draft<Payload>, action: patch.Action): HandlerResult => {
  const opts = { schema: PAYLOAD_Z, doc: state };
  const inverse = patch.apply(state, patch.snakeToCamel(action, opts), {
    snapshot: actions.snapshotDraft,
  });
  if (inverse.length === 0) return actions.NO_OP_RESULT;
  return {
    inverse: inverse.map((a) => patch.camelToSnake(a, opts)),
    targets: [patch.pathOf(action).join(".")],
  };
};

const handlers: Handlers = {
  create: (state, { task }) => {
    const { status: _, ...created } = task;
    Object.assign(state, created);
    return { inverse: [], targets: [task.key] };
  },
  set: (state, set) => applyPath(state, { type: "set", set }),
  remove: (state, remove) => applyPath(state, { type: "remove", remove }),
  insert: (state, insert) => applyPath(state, { type: "insert", insert }),
  move: (state, move) => applyPath(state, { type: "move", move }),
};

/**
 * Applies actions to a task in order.
 * @throws {Error} if the task cannot hold an action, such as a set through a number.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the task that a create action carries, or undefined for other actions. */
export const createOf = (action: Action): Payload | undefined => {
  if (action.type !== "create") return undefined;
  const { status: _, ...created } = action.create.task;
  return created;
};
