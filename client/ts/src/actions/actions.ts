// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { current, type Draft, isDraft, produce } from "immer";
import { z } from "zod";

/**
 * Result returned by a single per-variant action handler. `inverse` is the
 * action sequence the undo stack should replay to revert the handler's effect.
 * `targets` is the set of document keys the handler touched, used by consumers
 * to invalidate undoables targeting overlapping resources when a remote
 * session emits a competing action.
 */
export interface HandlerResult<A> {
  inverse: A[];
  targets: readonly string[];
}

/**
 * NO_OP_RESULT is the zero-value HandlerResult. Return it from a handler that
 * declined to mutate state (e.g. lookup miss) so it neither contributes to the
 * undo stack nor invalidates undoables targeting the same resource.
 */
export const NO_OP_RESULT: HandlerResult<never> = { inverse: [], targets: [] };

/**
 * Result returned by a reduceAll application: the new state, the inverse
 * sequence that reverts the whole batch, and the union of all targets touched.
 */
export interface ReduceAllResult<S, A> {
  next: S;
  inverse: A[];
  targets: readonly string[];
}

/**
 * Returns a plain-object snapshot of an Immer draft so the result is safe to
 * embed in an action stored on the undo stack. Returns non-draft inputs
 * unchanged. The non-draft passthrough exists because when a reducer applies
 * multiple actions in one produce(), an earlier action's wholesale
 * assignment leaves the slot as a plain object, so the next action would
 * crash if it called `current` unconditionally.
 */
export const snapshotDraft = <T>(v: T): T => (isDraft(v) ? current(v as Draft<T>) : v);

/**
 * createReduceAll lifts a per-variant `reduceOne` switch into a batched
 * reducer. The returned function applies each action against an Immer draft of
 * `state` in order, accumulates the inverses in reverse application order so a
 * single undo replays them correctly, and unions the touched targets.
 */
export const createReduceAll =
  <S, A>(reduceOne: (draft: Draft<S>, action: A) => HandlerResult<A>) =>
  (state: S, actions: A[]): ReduceAllResult<S, A> => {
    const inverse: A[] = [];
    const targetsSet = new Set<string>();
    const next = produce(state, (draft) => {
      for (const action of actions) {
        const result = reduceOne(draft, action);
        if (result.inverse.length > 0) inverse.unshift(...result.inverse);
        for (const t of result.targets) targetsSet.add(t);
      }
    });
    return { next, inverse, targets: Array.from(targetsSet) };
  };

/**
 * scopedZ builds the schema for envelopes broadcast on the sy_<service>_set
 * cluster signals channel. The framer's JSON codec runs snakeToCamel before
 * handing the value to the schema, so `dispatchKey` and `seq` are camelCase
 * here even though the server emits them as snake_case. `seq` defaults to 0 so
 * frames from servers that predate the field stay parseable.
 */
export const scopedZ = <K extends z.ZodType, A extends z.ZodType>(
  keyZ: K,
  actionZ: A,
) =>
  z.object({
    key: keyZ,
    dispatchKey: z.string(),
    seq: z.number().int().nonnegative().default(0),
    actions: actionZ.array(),
  });

/**
 * dispatchReqZ builds the request body schema for the per-service dispatch
 * endpoint. Stays camelCase; the JSON codec runs camelToSnake on encode so the
 * wire ends up snake_case without per-field conversion here.
 */
export const dispatchReqZ = <K extends z.ZodType, A extends z.ZodType>(
  keyZ: K,
  actionZ: A,
) =>
  z.object({
    key: keyZ,
    dispatchKey: z.string(),
    actions: actionZ.array(),
  });
