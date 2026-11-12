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
} from "@/arc/directory/actions.gen";
import { type Directory } from "@/arc/directory/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  name: z.string(),
  parent: zod.nullToUndefined(z.string()),
});

const applyPath = (state: Draft<Directory>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { directory }) => actions.replace(state, directory),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a directory in order. Path actions edit only the directory's name
 * and parent.
 * @throws {Error} if the directory cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the directory that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Directory | undefined =>
  action.type === "create" ? action.create.directory : undefined;
