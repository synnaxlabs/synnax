// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type patch, record, zod } from "@synnaxlabs/x";
import { type Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/device/actions.gen";
import { type Device } from "@/device/types.gen";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  name: z.string(),
  model: z.string(),
  location: z.string(),
  configured: z.boolean(),
  properties: record.unknownZ(),
  group: zod.nullToUndefined(z.string()),
});

const applyPath = (state: Draft<Device>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { device: { status: _, ...device } }) =>
    actions.replace(state, device),
  ...actions.createPathHandlers(applyPath),
  setRack: (state, { rack, parent }) => {
    state.rack = rack;
    if (parent == null) delete state.parent;
    else state.parent = parent;
    delete state.group;
    return { inverse: [], targets: ["rack"] };
  },
};

/**
 * Applies actions to a device in order. Path actions edit only the device's name,
 * model, location, configured flag, properties, and group.
 * @throws {Error} if the device cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the device that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Omit<Device, "status"> | undefined => {
  if (action.type !== "create") return undefined;
  const { status: _, ...device } = action.create.device;
  return device;
};
