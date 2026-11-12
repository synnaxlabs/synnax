// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type patch } from "@synnaxlabs/x";
import { type Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/secret/actions.gen";
import { type Secret } from "@/secret/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({ name: z.string() });

const applyPath = (state: Draft<Secret>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { secret }) => actions.replace(state, secret),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a secret in order. Path actions edit only the secret's name.
 * @throws {Error} if the secret cannot hold an action, such as a set of its value.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the secret that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Secret | undefined =>
  action.type === "create" ? action.create.secret : undefined;
