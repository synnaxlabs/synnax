// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control, patch, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { keyZ, localKeyZ, nameZ, operationZ } from "@/channel/types.gen";
import { group } from "@/group";
import { node } from "@/node";

/** The channel as the Core stores it, which a create action carries. */
export const storedZ = z.object({
  name: nameZ,
  leaseholder: node.keyZ,
  dataType: z.string(),
  isIndex: z.boolean(),
  localKey: localKeyZ,
  localIndex: localKeyZ,
  virtual: z.boolean(),
  concurrency: control.concurrencyZ,
  internal: z.boolean(),
  operations: zod.nullToUndefined(operationZ.array()),
  expression: z.string(),
  group: zod.nullToUndefined(group.keyZ),
});

export type Stored = z.infer<typeof storedZ>;

/**
 * Create replaces the channel with the given created state. Emitted by the Core when a
 * channel is created; clients never dispatch it.
 */
export const createPayloadZ = z.object({ channel: storedZ });

export type CreatePayload = z.infer<typeof createPayloadZ>;

/** Rename renames the channel. */
export const renamePayloadZ = z.object({ name: nameZ });

export type RenamePayload = z.infer<typeof renamePayloadZ>;

/** Action is an edit of a channel, in the form the Core takes. */
export type Action =
  | { type: "create"; create: CreatePayload }
  | { type: "set"; set: patch.Set }
  | { type: "remove"; remove: patch.Remove }
  | { type: "insert"; insert: patch.Insert }
  | { type: "move"; move: patch.Move }
  | { type: "rename"; rename: RenamePayload };

export const actionZ: z.ZodType<Action> = z.discriminatedUnion("type", [
  z.object({ type: z.literal("create"), create: createPayloadZ }),
  z.object({ type: z.literal("set"), set: patch.setZ }),
  z.object({ type: z.literal("remove"), remove: patch.removeZ }),
  z.object({ type: z.literal("insert"), insert: patch.insertZ }),
  z.object({ type: z.literal("move"), move: patch.moveZ }),
  z.object({ type: z.literal("rename"), rename: renamePayloadZ }),
]);

/** Set replaces the value at a path in the channel. */
export const set = (payload: z.input<typeof patch.setZ>): Action => ({
  type: "set",
  set: zod.parse(patch.setZ, payload, { label: "channel set action payload" }),
});

export const rename = (payload: z.input<typeof renamePayloadZ>): Action => ({
  type: "rename",
  rename: zod.parse(renamePayloadZ, payload, {
    label: "channel rename action payload",
  }),
});

export const scopedActionZ = actions.scopedZ(keyZ, actionZ);

export const dispatchReqZ = actions.dispatchReqZ(keyZ, actionZ);

export const dispatchResZ = actions.dispatchResZ(actionZ);
