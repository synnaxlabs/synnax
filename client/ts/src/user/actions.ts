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
} from "@/user/actions.gen";
import { type User } from "@/user/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
// Only the Core sets root_user.
const documentZ = z.object({
  name: z.string(),
  role: z.string(),
  rootUser: z.boolean(),
});

const applyPath = (state: Draft<User>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { user }) => actions.replace(state, user),
  ...actions.createPathHandlers(applyPath),
  // The cached user holds no username.
  setUsername: () => actions.NO_OP_RESULT,
};

/**
 * Applies actions to a user in order. Path actions edit only the user's name and role,
 * and the root flag that only the Core sets.
 * @throws {Error} if the user cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the user that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): User | undefined =>
  action.type === "create" ? action.create.user : undefined;
