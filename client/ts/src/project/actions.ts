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
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/project/actions.gen";
import { type Project } from "@/project/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  name: z.string(),
  group: zod.nullToUndefined(z.string()),
});

const applyPath = (state: Draft<Project>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { project }) => actions.replace(state, project),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a project in order. Path actions edit only the project's name and
 * group.
 * @throws {Error} if the project cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the project that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Project | undefined =>
  action.type === "create" ? action.create.project : undefined;
