// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch, type record, zod } from "@synnaxlabs/x";
import { current, type Draft, isDraft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/calculation/actions.gen";
import { type Calculation, operationZ } from "@/calculation/types.gen";
import { channel } from "@/channel";

// The part of a calculation that path actions edit. The Core sets reads from each save
// and sends them as a set. It checks shape but not validity: the Core validates.
const documentZ = z
  .object({
    expression: z.string(),
    operations: operationZ.array(),
    primary: channel.keyZ.optional(),
    reads: channel.keyZ.array(),
  })
  .strict();

const applyPath = (state: Draft<Calculation>, action: patch.Action): HandlerResult => {
  // A plain copy keeps draft proxies out of the document and the inverse.
  const plain = isDraft(state) ? current(state) : state;
  const doc: record.Unknown = structuredClone({
    expression: plain.expression,
    operations: plain.operations,
    primary: plain.primary,
    reads: plain.reads,
  });
  const inverse = patch.apply(doc, action);
  if (inverse.length === 0) return actions.NO_OP_RESULT;
  const next = zod.parse(documentZ, doc, { label: "calculation" });
  const path = patch.pathOf(action);
  switch (path[0]) {
    case "expression":
      state.expression = next.expression;
      break;
    case "operations":
      state.operations = next.operations;
      break;
    case "primary":
      state.primary = next.primary;
      break;
    default:
      state.reads = next.reads;
  }
  return { inverse, targets: [path.join(".")] };
};

const handlers: Handlers = {
  create: (state, { calculation }) => {
    Object.assign(state, calculation);
    return { inverse: [], targets: [calculation.key] };
  },
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a calculation in order. Path actions edit only the expression,
 * the operations, the primary, and the reads.
 * @throws {Error} if the calculation cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/**
 * Converts an action from the camelCase form of the cached calculation into the
 * snake_case form that the Core stores.
 */
export const encode = (state: Calculation, action: Action): Action =>
  action.type === "create"
    ? action
    : patch.camelToSnake(action, { schema: documentZ, data: state });

/** @returns the calculation that a create action carries, or undefined otherwise. */
export const createOf = (action: Action): Calculation | undefined =>
  action.type === "create" ? action.create.calculation : undefined;
