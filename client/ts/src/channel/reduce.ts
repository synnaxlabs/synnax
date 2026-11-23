// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type patch, zod } from "@synnaxlabs/x";
import { current, type Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import { type Action, rename, type Stored } from "@/channel/actions";
import { type Key, operationZ, type Payload, payloadZ } from "@/channel/types.gen";
import { group } from "@/group";

const keyOf = (leaseholder: number, localKey: number): Key =>
  ((leaseholder << 20) | localKey) >>> 0;

const fromStored = ({ localKey, localIndex, ...rest }: Stored): Payload =>
  zod.parse(
    payloadZ,
    {
      ...rest,
      key: keyOf(rest.leaseholder, localKey),
      index: localIndex === 0 ? 0 : keyOf(rest.leaseholder, localIndex),
      // A create carries the expression of the calculation that writes the channel.
      calculated: rest.virtual && rest.expression !== "",
    },
    { label: "created channel" },
  );

/** @returns the channel that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Payload | undefined =>
  action.type === "create" ? fromStored(action.create.channel) : undefined;

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  dataType: payloadZ.shape.dataType,
  expression: z.string(),
  operations: operationZ.array(),
  group: zod.nullToUndefined(group.keyZ),
});

const applyPath = (state: Draft<Payload>, action: patch.Action) =>
  actions.applyField(state, action, documentZ);

/**
 * Applies actions to a channel in order. Path actions edit only the channel's data
 * type, expression, operations, and group.
 * @throws {Error} if the channel cannot hold an action, such as a set of its key.
 */
export const reduceAll = actions.createReduceAll<Payload, Action>((state, action) => {
  switch (action.type) {
    case "create": {
      // A create holds no calculation field: a set after it carries them.
      const { expression, operations } = current(state) as Payload;
      const created = { ...action.create.channel, expression, operations };
      return actions.replace(state, fromStored(created));
    }
    case "rename": {
      const prev = state.name;
      if (prev === action.rename.name) return actions.NO_OP_RESULT;
      state.name = action.rename.name;
      return { inverse: [rename({ name: prev })], targets: ["name"] };
    }
    default: {
      const result = applyPath(state, action);
      // The expression of a calculation reaches the channel after its create.
      const calculated = state.virtual && state.expression !== "";
      if (calculated === state.calculated) return result;
      state.calculated = calculated;
      return { ...result, targets: [...result.targets, "calculated"] };
    }
  }
});
