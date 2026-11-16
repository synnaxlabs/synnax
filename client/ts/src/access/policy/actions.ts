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

import { access } from "@/access/action";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/access/policy/actions.gen";
import { type Policy } from "@/access/policy/types.gen";
import { actions } from "@/actions";
import { resource } from "@/resource";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  name: z.string(),
  objects: resource.idZ.array(),
  actions: access.actionZ.array(),
  role: z.string(),
});

const applyPath = (state: Draft<Policy>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { policy }) => actions.replace(state, policy),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a policy in order. Path actions edit only the policy's name,
 * objects, actions, and role.
 * @throws {Error} if the policy cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the policy that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Policy | undefined =>
  action.type === "create" ? action.create.policy : undefined;
