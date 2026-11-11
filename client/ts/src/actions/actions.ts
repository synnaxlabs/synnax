// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, deep, patch, type record, zod } from "@synnaxlabs/x";
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
 * Returns a plain-object snapshot of an Immer draft so the result is safe to embed in
 * an action stored on the undo stack. Returns non-draft inputs unchanged. The non-draft
 * passthrough exists because when a reducer applies multiple actions in one produce(),
 * an earlier action's wholesale assignment leaves the slot as a plain object, so the
 * next action would crash if it called `current` unconditionally.
 */
export const snapshotDraft = <T>(v: T): T => (isDraft(v) ? current(v as Draft<T>) : v);

const applyInverse = <S extends object>(
  state: Draft<S>,
  action: patch.Action,
  schema: z.ZodType,
): patch.Action[] => {
  const opts = { schema, data: state };
  const inverse = patch.apply(
    state as record.Unknown,
    patch.snakeToCamel(action, opts),
    { snapshot: snapshotDraft },
  );
  return inverse.map((a) => patch.camelToSnake(a, opts));
};

/**
 * Applies a path action to a draft of a cached document. The action holds the form the
 * Core stores, with snake_case keys, and the draft holds the camelCase form that the
 * client decodes. The result targets the path that the action changes, and holds the
 * inverse actions in the form the Core stores.
 * @param schema - The document's schema, which names the keys that keep their case.
 * @throws {Error} if the document cannot hold the action.
 */
export const applyPath = <S extends object>(
  state: Draft<S>,
  action: patch.Action,
  schema: z.ZodType,
): HandlerResult<patch.Action> => {
  const inverse = applyInverse(state, action, schema);
  if (inverse.length === 0) return NO_OP_RESULT;
  return { inverse, targets: [patch.pathOf(action).join(".")] };
};

/**
 * Applies a path action to an entry of the map at root in a draft of a cached document,
 * such as one element config of a schematic. The result targets that entry.
 * @param root - The path of the map.
 * @param schema - The document's schema, which names the keys that keep their case.
 * @throws {Error} if the action's path does not reach an entry of the map, or if the
 * document cannot hold the action.
 */
export const applyEntryPath = <S extends object>(
  state: Draft<S>,
  action: patch.Action,
  root: string[],
  schema: z.ZodType,
): HandlerResult<patch.Action> => {
  const path = patch.pathOf(action);
  if (path.length <= root.length || root.some((field, i) => path[i] !== field))
    throw new Error(`${path.join(".")} is not inside an entry of ${root.join(".")}`);
  const inverse = applyInverse(state, action, schema);
  if (inverse.length === 0) return NO_OP_RESULT;
  return { inverse, targets: [path[root.length]] };
};

/**
 * Applies a path action to a draft of a cached record whose path actions edit only the
 * top-level fields of a document, then parses the field that the action changes. The
 * action holds the snake_case form that the Core stores. A field that parses to
 * undefined is removed.
 * @param document - The schema of the fields that path actions edit, which also names
 * the keys that keep their case.
 * @throws {Error} if the path does not start at a field of the document, or if the
 * field does not parse after the action.
 */
export const applyField = <S extends object>(
  state: Draft<S>,
  action: patch.Action,
  document: z.ZodObject,
): HandlerResult<patch.Action> => {
  const path = patch.pathOf(action);
  const [field] = caseconv.snakeToCamelPath(path.slice(0, 1), {
    schema: document,
  }).path;
  const fieldZ: z.ZodType | undefined =
    field == null || !Object.hasOwn(document.shape, field)
      ? undefined
      : document.shape[field];
  if (fieldZ == null) throw new Error(`no editable field at "${path.join(".")}"`);
  const result = applyPath(state, action, document);
  if (result.inverse.length === 0) return result;
  const draft = state as record.Unknown;
  const value = snapshotDraft(draft[field]);
  const next: unknown = zod.parse(fieldZ, value, { label: field });
  if (next === undefined) delete draft[field];
  // The parse copies the field but shares its frozen values, so writing back an equal
  // copy would leave a later action in the batch unable to edit them.
  else if (!deep.equal(next, value)) draft[field] = next;
  return result;
};

/**
 * Replaces a draft of a cached record with the record that a create carries. A create
 * has no inverse.
 */
export const replace = <S extends record.Keyed<record.Key>>(
  state: Draft<S>,
  created: S,
): HandlerResult<never> => {
  const draft = state as record.Unknown;
  for (const key of Object.keys(draft)) if (!(key in created)) delete draft[key];
  Object.assign(draft, created);
  return { inverse: [], targets: [String(created.key)] };
};

const PATH_TYPES: ReadonlySet<string> = new Set(["set", "remove", "insert", "move"]);

/**
 * @returns whether the action is a path action that changes one of the top-level
 * fields, named by wire name.
 */
export const changesField = (action: { type: string }, fields: readonly string[]) =>
  PATH_TYPES.has(action.type) &&
  fields.includes(patch.pathOf(action as patch.Action)[0]);

/**
 * Lists a set of each field that differs between base and next. Each set holds the
 * whole field, with its path and value in the snake_case form that the Core stores.
 * @param schema - The schema of base and next, which names the keys that keep their
 * case.
 * @param base - The stored values, or undefined to set each field that next holds.
 */
export const diffFields = <F extends string>(
  schema: z.ZodType,
  base: Partial<Record<F, unknown>> | undefined,
  next: Partial<Record<F, unknown>>,
  fields: F[],
): Array<{ type: "set"; set: patch.Set }> =>
  fields.flatMap((field) => {
    if (deep.equal(base?.[field], next[field])) return [];
    const { path, convert } = caseconv.camelToSnakePath([field], { schema });
    return [{ type: "set", set: { path, value: convert(next[field]), guards: [] } }];
  });

/**
 * createPathHandlers builds the set, remove, insert, and move handlers of a document's
 * action union from one function that applies a path action.
 */
export const createPathHandlers = <D, A>(
  apply: (state: D, action: patch.Action) => HandlerResult<A>,
) => ({
  set: (state: D, set: patch.Set) => apply(state, { type: "set", set }),
  remove: (state: D, remove: patch.Remove) => apply(state, { type: "remove", remove }),
  insert: (state: D, insert: patch.Insert) => apply(state, { type: "insert", insert }),
  move: (state: D, move: patch.Move) => apply(state, { type: "move", move }),
});

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

/** The payload of a dispatch entry in a batch. The entry carries no dispatch key. */
export interface DispatchPayload<K, A> {
  key: K;
  actions: A[];
}

/**
 * dispatchResZ builds the response body schema for the per-service dispatch endpoint:
 * the actions the Core added after the dispatched ones, such as a set of a field that
 * only the Core computes. A service that adds none sends an empty body.
 */
export const dispatchResZ = <A extends z.ZodType>(actionZ: A) =>
  z.object({ actions: actionZ.array().default([]) });
