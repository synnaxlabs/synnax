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
} from "@/icd/actions.gen";
import { type ICD, icdZ } from "@/icd/types.gen";

const applyPath = (state: Draft<ICD>, action: patch.Action): HandlerResult =>
  actions.applyPath(state, action, icdZ);

const handlers: Handlers = {
  create: (state, { icd }) => {
    Object.assign(state, icd);
    return { inverse: [], targets: [icd.key] };
  },
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to an ICD in order.
 * @throws {Error} if the ICD cannot hold an action, such as a set through a number.
 */
export const reduceAll = createReduceAll(handlers);

/**
 * Converts an action from the camelCase form of the cached ICD into the snake_case form
 * that the Core stores.
 */
export const encode = (state: ICD, action: Action): Action =>
  action.type === "create"
    ? action
    : patch.camelToSnake(action, { schema: icdZ, data: state });

/** @returns the ICD that a create action carries, or undefined for other actions. */
export const createOf = (action: Action): ICD | undefined =>
  action.type === "create" ? action.create.icd : undefined;
