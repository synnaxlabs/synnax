// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type patch, zod } from "@synnaxlabs/x";
import type { Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import { type Action, rename, type Stored } from "@/channel/actions";
import { type Key, operationZ, type Payload, payloadZ } from "@/channel/types.gen";
import { group } from "@/group";

const keyOf = (leaseholder: number, localKey: number): Key =>
  ((leaseholder << 20) | localKey) >>> 0;

const calculatedOf = ({
  virtual,
  expression,
}: Pick<Payload, "virtual" | "expression">) => virtual && expression !== "";

const fromStored = ({ localKey, localIndex, ...rest }: Stored): Payload =>
  zod.parse(
    payloadZ,
    {
      ...rest,
      key: keyOf(rest.leaseholder, localKey),
      index: localIndex === 0 ? 0 : keyOf(rest.leaseholder, localIndex),
      calculated: calculatedOf(rest),
    },
    { label: "created channel" },
  );

// calculatedOf reads only the expression, which only a subject who reads the
// calculation receives. Another subject holds calculated with no expression.
const EXPRESSION = ["expression"] as const;

/** @returns the channel that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Payload | undefined =>
  action.type === "create" ? fromStored(action.create.channel) : undefined;

// The fields that path actions edit, checked for shape only. The Core validates them.
const documentZ = z.object({
  dataType: payloadZ.shape.dataType,
  expression: z.string(),
  operations: operationZ.array(),
  group: zod.nullToUndefined(group.keyZ),
  description: z.string(),
});

const applyPath = (state: Draft<Payload>, action: patch.Action) =>
  actions.applyField(state, action, documentZ);

/**
 * Applies actions to a channel in order. Path actions edit only the channel's data
 * type, expression, operations, group, and description.
 * @throws {Error} if the channel cannot hold an action, such as a set of its key.
 */
export const reduceAll = actions.createReduceAll<Payload, Action>((state, action) => {
  switch (action.type) {
    case "create": {
      const { expression, operations, calculated } = state;
      const result = actions.replace(state, fromStored(action.create.channel));
      // A create holds no calculation field, so the cached ones stay.
      Object.assign(state, { expression, operations, calculated });
      return result;
    }
    case "rename": {
      const prev = state.name;
      if (prev === action.rename.name) return actions.NO_OP_RESULT;
      state.name = action.rename.name;
      return { inverse: [rename({ name: prev })], targets: ["name"] };
    }
    default: {
      const result = applyPath(state, action);
      if (!actions.changesField(action, EXPRESSION)) return result;
      const calculated = calculatedOf(state);
      if (calculated === state.calculated) return result;
      state.calculated = calculated;
      return { ...result, targets: [...result.targets, "calculated"] };
    }
  }
});
