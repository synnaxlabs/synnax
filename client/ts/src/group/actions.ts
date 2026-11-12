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
} from "@/group/actions.gen";
import { type Group } from "@/group/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({ name: z.string() });

const applyPath = (state: Draft<Group>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { group }) => actions.replace(state, group),
  ...actions.createPathHandlers(applyPath),
  setParent: (state, { parent }) => {
    if (parent == null) delete state.parent;
    else state.parent = parent;
    return { inverse: [], targets: ["parent"] };
  },
};

/**
 * Applies actions to a group in order. Path actions edit only the group's name.
 * @throws {Error} if the group cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the group that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Group | undefined =>
  action.type === "create" ? action.create.group : undefined;
