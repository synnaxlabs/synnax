// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch, type record } from "@synnaxlabs/x";
import { type Draft } from "immer";
import { type z } from "zod";

import { actions } from "@/actions";
import { type HandlerResult } from "@/task/actions.gen";
import {
  type Action,
  type CreatePayload,
  type Payload,
  payloadZ,
  updateZ,
} from "@/task/types.gen";

// The cache decodes every task with the generic schema, so actions convert with it.
const PAYLOAD_Z = payloadZ();

const applyPath = (state: Draft<Payload>, action: patch.Action): HandlerResult =>
  actions.applyPath(state, action, PAYLOAD_Z);

const create = (state: Draft<Payload>, { task }: CreatePayload): HandlerResult => {
  const { status: _, ...created } = task;
  Object.assign(state, created);
  return { inverse: [], targets: [task.key] };
};

/** The schemas of the config of each task type, keyed by the task type. */
export type ConfigSchemas = Record<
  string,
  patch.StructSchemas<z.ZodType, z.ZodType<record.Unknown>>
>;

const RECORD_UPDATE_SCHEMAS = { u: updateZ(), t: PAYLOAD_Z };

type UpdateSchemas = typeof RECORD_UPDATE_SCHEMAS;

/**
 * Builds a function that applies actions to a task in order. An update merges into the
 * config by the schemas of the task's type, and replaces the config whole when the type
 * has no schemas.
 *
 * The function throws an Error if the task cannot hold an action, such as a set through
 * a number, and a patch.StaleError if a guard of an update does not hold.
 */
export const createReducer = (configs: ConfigSchemas) => {
  const updates = new Map<string, UpdateSchemas>(
    Object.entries(configs).map(([type, config]) => [
      type,
      { u: updateZ(config), t: PAYLOAD_Z },
    ]),
  );
  return actions.createReduceAll<Payload, Action>((state, action) => {
    switch (action.type) {
      case "create":
        return create(state, action.create);
      case "update":
        return actions.applyUpdate(
          state,
          action.update,
          updates.get(state.type) ?? RECORD_UPDATE_SCHEMAS,
        );
      default:
        return applyPath(state, action);
    }
  });
};

/**
 * Converts an action from the camelCase form of the cached task into the snake_case
 * form that the Core stores.
 */
export const encode = (state: Payload, action: Action): Action =>
  action.type === "create" || action.type === "update"
    ? action
    : patch.camelToSnake(action, { schema: PAYLOAD_Z, data: state });

/** @returns the task that a create action carries, or undefined for other actions. */
export const createOf = (action: Action): Payload | undefined => {
  if (action.type !== "create") return undefined;
  const { status: _, ...created } = action.create.task;
  return created;
};
