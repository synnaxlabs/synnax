// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, patch, type record, zod } from "@synnaxlabs/x";
import { type Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/label/actions.gen";
import { type Label } from "@/label/types.gen";

// The part of a label that path actions edit. It checks shape but not validity: the
// Core validates, and a form holds an invalid value unsent while the cache shows it.
const documentZ = z.object({ name: z.string(), color: color.colorZ }).strict();

const applyPath = (state: Draft<Label>, action: patch.Action): HandlerResult => {
  // The copy of the color keeps draft proxies out of the inverse.
  const doc: record.Unknown = { name: state.name, color: [...state.color] };
  const inverse = patch.apply(doc, action);
  if (inverse.length === 0) return actions.NO_OP_RESULT;
  const next = zod.parse(documentZ, doc, { label: "label" });
  const path = patch.pathOf(action);
  if (path[0] === "name") state.name = next.name;
  else state.color = next.color;
  return { inverse, targets: [path.join(".")] };
};

const handlers: Handlers = {
  create: (state, { label }) => {
    Object.assign(state, label);
    return { inverse: [], targets: [label.key] };
  },
  set: (state, set) => applyPath(state, { type: "set", set }),
  remove: (state, remove) => applyPath(state, { type: "remove", remove }),
  insert: (state, insert) => applyPath(state, { type: "insert", insert }),
  move: (state, move) => applyPath(state, { type: "move", move }),
};

/**
 * Applies actions to a label in order. Path actions edit only the label's name and
 * color.
 * @throws {Error} if the label cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

/** @returns the label that a create action carries, or undefined for other actions. */
export const createOf = (action: Action): Label | undefined =>
  action.type === "create" ? action.create.label : undefined;
