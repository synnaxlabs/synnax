// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { keyZ, nameZ, updateZ } from "@/channel/types.gen";

/**
 * Rename renames the channel. The Core also renames the channel's storage when the
 * transaction commits.
 */
export const renamePayloadZ = z.object({ name: nameZ });

export type RenamePayload = z.infer<typeof renamePayloadZ>;

/** Update changes the fields of the channel that it holds, when its guards hold. */
export const updatePayloadZ = patch.updateZ(updateZ);

export type UpdatePayload = z.infer<typeof updatePayloadZ>;

/** Action is an edit of a channel, in the form the Core takes. */
export type Action =
  | { type: "update"; update: UpdatePayload }
  | { type: "set"; set: patch.Set }
  | { type: "rename"; rename: RenamePayload };

export const actionZ: z.ZodType<Action> = z.discriminatedUnion("type", [
  z.object({ type: z.literal("update"), update: updatePayloadZ }),
  z.object({ type: z.literal("set"), set: patch.setZ }),
  z.object({ type: z.literal("rename"), rename: renamePayloadZ }),
]);

export const update = (payload: z.input<typeof updatePayloadZ>): Action => ({
  type: "update",
  update: zod.parse(updatePayloadZ, payload, {
    label: "channel update action payload",
  }),
});

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

export const dispatchReqZ = actions.dispatchReqZ(keyZ, actionZ);

export const dispatchResZ = actions.dispatchResZ(actionZ);
