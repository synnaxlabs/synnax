// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type patch, zod } from "@synnaxlabs/x";
import { type Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import { ranger } from "@/ranger";
import { status } from "@/status";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/view/actions.gen";
import { type Type, type View } from "@/view/types.gen";

// The fields that path actions edit for each type of view, checked for shape only. The
// Core validates them.
const DOCUMENTS: Record<Type, z.ZodObject> = {
  range: z.object({ name: z.string(), where: zod.nullToUndefined(ranger.whereZ) }),
  status: z.object({ name: z.string(), where: zod.nullToUndefined(status.whereZ) }),
};

const applyPath = (state: Draft<View>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, DOCUMENTS[state.type]);

const handlers: Handlers = {
  create: (state, { view }) => actions.replace(state, view),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a view in order. Path actions edit only the view's name and
 * filter.
 * @throws {Error} if the view cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the view that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): View | undefined =>
  action.type === "create" ? action.create.view : undefined;
