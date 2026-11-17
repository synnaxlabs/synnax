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

import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/access/role/actions.gen";
import { type Role } from "@/access/role/types.gen";
import { actions } from "@/actions";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({ name: z.string(), description: z.string() });

const applyPath = (state: Draft<Role>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { role }) => actions.replace(state, role),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a role in order. Path actions edit only the role's name and
 * description.
 * @throws {Error} if the role cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the role that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Role | undefined =>
  action.type === "create" ? action.create.role : undefined;
