// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  caseconv,
  color,
  patch,
  type record,
  TimeRange,
  TimeStamp,
  zod,
} from "@synnaxlabs/x";
import { type Draft } from "immer";
import { z } from "zod";

import { actions } from "@/actions";
import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/ranger/actions.gen";
import { type Key, keyZ, type Payload } from "@/ranger/types.gen";

// The part of a range that path actions edit. It checks shape but not validity: the
// Core validates, and a form holds an invalid value unsent while the cache shows it.
const documentZ = z
  .object({
    name: z.string(),
    timeRange: TimeRange.z,
    color: color.colorZ.nullish(),
    labels: z.string().array(),
    parent: z.string().nullish(),
  })
  .strict();

// Actions hold the wire form of a range: snake_case keys and nanosecond timestamps.
// Immer cannot draft the cached range's TimeRange, so each action applies to the wire
// form of the range, which then decodes back.
const applyPath = (state: Draft<Payload>, action: patch.Action): HandlerResult => {
  const { name, timeRange, color, parent } = state;
  // The copy of the labels keeps draft proxies out of the inverse. A field the range
  // lacks stays out of the document, so the inverse of its set is a remove.
  const fields: record.Unknown = { name, timeRange, labels: [...state.labels] };
  if (color != null) fields.color = color;
  if (parent != null) fields.parent = parent;
  const doc: record.Unknown = caseconv.camelToSnake(fields, { schema: documentZ });
  const inverse = patch.apply(doc, action);
  if (inverse.length === 0) return actions.NO_OP_RESULT;
  const next = zod.parse(documentZ, caseconv.snakeToCamel(doc, { schema: documentZ }), {
    label: "range",
  });
  const path = patch.pathOf(action);
  // Only the field the action reaches changes, so the others keep their identity.
  switch (path[0]) {
    case "name":
      state.name = next.name;
      break;
    case "time_range":
      state.timeRange = next.timeRange;
      break;
    case "labels":
      state.labels = next.labels;
      break;
    case "parent":
      if (next.parent == null) delete state.parent;
      else state.parent = next.parent;
      break;
    case "color":
      if (next.color == null) delete state.color;
      else state.color = next.color;
  }
  return { inverse, targets: [path.join(".")] };
};

const handlers: Handlers = {
  // A fetch can cache a label write that commits after the create, before the create
  // echo arrives. The echo keeps the cached labels so it does not undo that write. It
  // keeps hasChildren too, which a create always carries as false.
  create: (state, { range }) => {
    for (const key of Object.keys(state) as Array<keyof Payload>)
      if (!(key in range)) delete state[key];
    Object.assign(state, {
      ...range,
      labels: state.labels,
      hasChildren: state.hasChildren,
    });
    return { inverse: [], targets: [range.key] };
  },
  ...actions.createPathHandlers(applyPath),
};

/**
 * Applies actions to a range in order. Path actions edit only the range's name, time
 * range, color, labels, and parent. A create keeps the labels and hasChildren of the
 * range.
 * @throws {Error} if the range cannot hold an action, such as a set of its key.
 */
export const reduceAll = createReduceAll(handlers);

const nanoseconds = (time: TimeStamp): string => time.valueOf().toString();

// A number cannot hold a large timestamp such as TimeStamp.MAX, so a timestamp reaches
// the Core as a string of its exact nanoseconds.
const exactTimes = (action: patch.Action): patch.Action => {
  if (action.type !== "set" || action.set.path[0] !== "time_range") return action;
  const { set } = action;
  if (set.path.length > 1)
    return {
      ...action,
      set: { ...set, value: nanoseconds(TimeStamp.z.parse(set.value)) },
    };
  const { start, end } = TimeRange.z.parse(set.value);
  const value = { start: nanoseconds(start), end: nanoseconds(end) };
  return { ...action, set: { ...set, value } };
};

/**
 * Converts an action from the camelCase form of the cached range into the snake_case
 * form that the Core stores, with exact timestamps.
 */
export const encode = (state: Payload, action: Action): Action =>
  action.type === "create"
    ? action
    : exactTimes(patch.camelToSnake(action, { schema: documentZ, data: state }));

/** @returns the range that a create action carries, or undefined for other actions. */
export const createOf = (action: Action): Payload | undefined =>
  action.type === "create" ? action.create.range : undefined;

/**
 * @returns the parent that a create or a set of the parent places a range under, or
 * undefined for other actions.
 */
export const parentOf = (action: Action): Key | undefined => {
  if (action.type === "create") return action.create.range.parent;
  if (action.type !== "set" || action.set.path.join(".") !== "parent") return undefined;
  return keyZ.optional().parse(action.set.value ?? undefined);
};
