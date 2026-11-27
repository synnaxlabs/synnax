// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { patch } from "@synnaxlabs/x";
import type { Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import { channel } from "@/channel";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/labjack/device/actions.gen";
import { connectionZ, type Device, modelZ, portsZ } from "@/labjack/device/types.gen";
import { rack } from "@/rack";

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  rack: rack.keyZ,
  name: z.string(),
  serial: z.number(),
  connection: connectionZ,
  identifier: z.string(),
  readIndex: channel.keyZ.optional(),
  writeStateIndex: channel.keyZ.optional(),
  model: modelZ,
  ports: portsZ,
});

const applyPath = (state: Draft<Device>, action: patch.Action): HandlerResult =>
  actions.applyField(state, action, documentZ);

const handlers: Handlers = {
  create: (state, { device }) => actions.replace(state, device),
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a device in order. Path actions edit every field but the key.
 * @throws {Error} if the device cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the device that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Device | undefined =>
  action.type === "create" ? action.create.device : undefined;
