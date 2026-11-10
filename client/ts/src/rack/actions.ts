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
} from "@/rack/actions.gen";
import { type Payload } from "@/rack/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  name: z.string(),
  integrations: z.string().array(),
  version: zod.nullToUndefined(z.string()),
  group: zod.nullToUndefined(z.string()),
});

const applyPath = (state: Draft<Payload>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { rack: { status: _, ...rack } }) => actions.replace(state, rack),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a rack in order. Path actions edit only the rack's name,
 * integrations, version, and group.
 * @throws {Error} if the rack cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the rack that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Omit<Payload, "status"> | undefined => {
  if (action.type !== "create") return undefined;
  const { status: _, ...rack } = action.create.rack;
  return rack;
};
