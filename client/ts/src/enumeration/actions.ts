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
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/enumeration/actions.gen";
import { entryZ, type Enumeration } from "@/enumeration/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  name: z.string(),
  entries: entryZ.extend({ key: z.string() }).array(),
});

const applyPath = (state: Draft<Enumeration>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { enumeration }) => actions.replace(state, enumeration),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to an enumeration in order. Path actions edit only its name and its
 * entries.
 * @throws {Error} if the enumeration cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/**
 * Converts an action from the camelCase form of the cached enumeration into the
 * snake_case form that the Core stores.
 */
export const encode = (state: Enumeration, action: Action): Action =>
  action.type === "create"
    ? action
    : patch.camelToSnake(action, { schema: documentZ, data: state });

/** @returns the enumeration that a create action carries, or undefined for others. */
export const createOf = (action: Action): Enumeration | undefined =>
  action.type === "create" ? action.create.enumeration : undefined;
