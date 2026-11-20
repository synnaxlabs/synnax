// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, patch, type record, zod } from "@synnaxlabs/x";
import type { Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/color/actions.gen";
import type { Named } from "@/color/types.gen";

// The part of a named color that path actions edit. It checks shape, not validity: the
// Core validates, and a form holds an invalid value unsent while the cache shows it.
const documentZ = z
  .object({
    name: z.string(),
    light: color.colorZ,
    dark: color.colorZ.optional(),
    group: zod.nullToUndefined(z.string()),
    position: z.string().optional(),
  })
  .strict();

const applyPath = (state: Draft<Named>, action: patch.Action): HandlerResult => {
  // The copies of the colors keep draft proxies out of the inverse.
  const doc: record.Unknown = {
    name: state.name,
    light: [...state.light],
    ...(state.dark != null ? { dark: [...state.dark] } : {}),
    ...(state.group != null ? { group: state.group } : {}),
    ...(state.position != null ? { position: state.position } : {}),
  };
  const inverse = patch.apply(doc, action);
  if (inverse.length === 0) return actions.NO_OP_RESULT;
  const next = zod.parse(documentZ, doc, { label: "named color" });
  const path = patch.pathOf(action);
  if (path[0] === "name") state.name = next.name;
  else if (path[0] === "light") state.light = next.light;
  else if (path[0] === "group") state.group = next.group;
  else if (path[0] === "position") state.position = next.position;
  else state.dark = next.dark;
  return { inverse, targets: [path.join(".")] };
};

const handlers: Handlers = {
  create: (state, { named }) => {
    Object.assign(state, named);
    return { inverse: [], targets: [named.key] };
  },
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a named color in order. Path actions edit only its name, light
 * value, dark value, group, and position.
 * @throws {Error} if the named color cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/**
 * Converts an action from the camelCase form of the cached named color into the
 * snake_case form that the Core stores.
 */
export const encode = (state: Named, action: Action): Action =>
  action.type === "create"
    ? action
    : patch.camelToSnake(action, { schema: documentZ, data: state });

/** @returns the named color that a create action carries, or undefined for others. */
export const createOf = (action: Action): Named | undefined =>
  action.type === "create" ? action.create.named : undefined;
