// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type actions,
  DisconnectedError,
  query,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Status } from "@synnaxlabs/lyra/status";
import {
  deep,
  type destructor,
  patch,
  type record,
  TimeSpan,
  zod,
} from "@synnaxlabs/x";
import { useEffect, useState } from "react";
import { type z } from "zod";

import { Synnax } from "@/synnax";

/** The part of a domain client that a document form reads and edits through. */
export interface DocumentDomain<
  Key extends record.Key,
  Doc extends query.Data,
  Action = patch.Action,
> {
  watch: (key: Key) => query.Live<Doc>;
  /** Begins a transaction of actions on the cached document, with camelCase keys. */
  beginTransaction: (key: Key) => actions.Transaction<Action>;
}

/** How a document form sends each edit as an update of its values. */
export interface DocumentUpdates<Action, Update extends z.ZodType = z.ZodType> {
  /** The update schema of the form's values, which gives the guards of an update. */
  schema: Update;
  /** Builds the action that carries an update of the form's values. */
  action: (update: patch.Update<Update>) => Action;
}

/** Names the document that a document form edits. */
export interface DocumentQuery<Key extends record.Key> {
  key: Key;
}

/** How long a field holds its edits before it sends them as one dispatch. */
const COALESCE_WINDOW = TimeSpan.milliseconds(200);

/** The value that an edit writes at a path. */
interface Write<Action> {
  /** The action that writes the value. */
  action: Action;
  value: unknown;
  /** Whether the value passes its field schema. */
  valid: boolean;
}

/** The edits to one field that have not been sent. */
interface Edit<Action> {
  tx: actions.Transaction<Action>;
  /** What the edit writes at each path. */
  writes: Map<string, Write<Action>>;
  timer?: ReturnType<typeof setTimeout>;
}

/** Where a form path lands in the document. */
interface Target {
  /** The path of an action, naming each list entry by its key. */
  path: string[];
  /** The value at the path. */
  node: unknown;
  /**
   * The rest of the form path, below a list of entries without keys. Such a list is set
   * whole.
   */
  rest: string;
  /** Whether the path ends at a list entry. */
  entry: boolean;
}

const isEntry = (value: unknown): value is record.Keyed<string> =>
  value != null &&
  typeof value === "object" &&
  typeof (value as record.Unknown).key === "string";

const hasKey = (schema: z.core.$ZodType): boolean => {
  const def = zod.unwrap(schema)._zod.def as z.core.$ZodTypeDef & record.Unknown;
  if (def.type === "object") return "key" in (def.shape as record.Unknown);
  if (def.type === "union") return (def.options as z.core.$ZodType[]).every(hasKey);
  return false;
};

// The schema decides, since an empty keyed list holds no entry to show its keys.
const isKeyedList = (schema: z.ZodType, formPath: string, values: unknown): boolean => {
  const list = zod.getFieldSchema(schema, formPath, { optional: true, values });
  if (list == null) return false;
  const def = zod.unwrap(list)._zod.def as z.core.$ZodTypeDef & record.Unknown;
  return def.type === "array" && hasKey(def.element as z.core.$ZodType);
};

/**
 * Finds where a form path lands in the values.
 *
 * @returns null when the path goes through a keyed list entry that is not there, as
 * when another client removed it.
 */
const locate = (
  schema: z.ZodType,
  values: unknown,
  formPath: string,
): Target | null => {
  const parts = formPath.split(deep.SEPARATOR);
  const path: string[] = [];
  let node = values;
  let entry = false;
  let i = 0;
  while (i < parts.length) {
    if (Array.isArray(node)) {
      const list = node as unknown as record.Unknown;
      const best = deep.findBestKey(list, parts.slice(i));
      const found = best == null ? null : deep.defaultGetter(list, best[0]);
      if (best != null && isEntry(found)) {
        path.push(found.key);
        node = found;
        entry = true;
        i += best[1];
        continue;
      }
      if (isKeyedList(schema, parts.slice(0, i).join(deep.SEPARATOR), values))
        return null;
      return { path, node, rest: parts.slice(i).join(deep.SEPARATOR), entry: false };
    }
    if (node == null || typeof node !== "object") break;
    const best = deep.findBestKey(node as record.Unknown, parts.slice(i));
    if (best == null) break;
    const [key, consumed] = best;
    path.push(key);
    node = (node as record.Unknown)[key];
    entry = false;
    i += consumed;
  }
  if (i < parts.length) {
    path.push(...parts.slice(i));
    node = undefined;
    entry = false;
  }
  return { path, node, rest: "", entry };
};

// @returns the value at the target after value is written at the rest of its path.
const wholeAt = ({ node, rest }: Target, value: unknown): unknown => {
  if (rest === "") return value;
  const whole = deep.copy(node);
  deep.set(whole, rest, value);
  return whole;
};

const edited = (values: unknown, edit: (copy: unknown) => void): unknown => {
  const copy = deep.copy(values);
  edit(copy);
  return copy;
};

/** Turns the edits of a form into the actions that reach the Core. */
interface Codec<Action> {
  set: (values: unknown, target: Target, formPath: string, value: unknown) => Action;
  remove: (values: unknown, target: Target, formPath: string) => Action;
  insert: (
    values: unknown,
    list: string[],
    value: Form.Entry,
    after?: string,
  ) => Action;
  move: (values: unknown, list: string[], key: string, after?: string) => Action;
  /**
   * @returns the action that turns values into next, or null when they are equal. A
   * codec without diff writes each field that a write of the whole form changes.
   */
  diff?: (values: unknown, next: unknown) => Action | null;
}

// Writes path actions at the paths of the form's values below prefix.
const pathCodec = (schema: z.ZodType, prefix: string[]): Codec<patch.Action> => {
  const guards = (values: unknown, path: string[]) =>
    patch.guards(schema, path, values);
  const out = (action: patch.Action) => patch.prefix(action, prefix);
  return {
    set: (values, target, _, value) =>
      out({
        type: "set",
        set: {
          path: target.path,
          value: wholeAt(target, value),
          guards: guards(values, target.path),
        },
      }),
    remove: (values, { path, node, rest }) => {
      if (rest !== "") {
        const whole = deep.copy(node);
        deep.remove(whole, rest);
        return out({
          type: "set",
          set: { path, value: whole, guards: guards(values, path) },
        });
      }
      const parent = path.slice(0, -1);
      const key = path[path.length - 1];
      return out({
        type: "remove",
        remove: { path: parent, key, guards: guards(values, path) },
      });
    },
    insert: (values, path, value, after) =>
      out({
        type: "insert",
        insert: { path, value, after, guards: guards(values, path) },
      }),
    move: (values, path, key, after) =>
      out({ type: "move", move: { path, key, after, guards: guards(values, path) } }),
  };
};

const isComposite = (value: unknown): boolean =>
  value != null && typeof value === "object";

// Writes updates of the form's values. A list entry goes by its key.
const updateCodec = <Action, Update extends z.ZodType>({
  schema,
  action,
}: DocumentUpdates<Action, Update>): Codec<Action> => {
  const nested = (values: unknown, path: string[], leaf: unknown): Action =>
    action(patch.nest(schema, values, path, leaf));
  const toUpdate = (values: unknown, next: unknown): Action =>
    action(patch.diffUpdate(schema, values, next));
  return {
    set: (values, target, formPath, value) =>
      target.rest === "" && !isComposite(value)
        ? nested(values, target.path, value)
        : toUpdate(
            values,
            edited(values, (copy) => deep.set(copy, formPath, value)),
          ),
    remove: (values, { path, rest, entry }, formPath) => {
      if (rest !== "")
        return toUpdate(
          values,
          edited(values, (copy) => deep.remove(copy, formPath)),
        );
      if (!entry) return nested(values, path, null);
      return nested(values, path.slice(0, -1), { remove: [path[path.length - 1]] });
    },
    insert: (values, path, value, after) =>
      nested(values, path, { insert: [value], move: [{ key: value.key, after }] }),
    move: (values, path, key, after) =>
      nested(values, path, { move: [{ key, after }] }),
    diff: (values, next) => {
      const update = patch.diffUpdate(schema, values, next);
      const { value } = update as patch.Update<z.ZodType>;
      if (Object.keys(value as record.Unknown).length === 0) return null;
      return action(update);
    },
  };
};

// Writes the top-level fields of the form's values that an edit changes, whole.
const fieldsCodec = <Action>(
  action: (fields: record.Unknown) => Action,
): Codec<Action> => {
  const changed = (values: unknown, next: unknown): record.Unknown => {
    const prev = values as record.Unknown;
    const cur = next as record.Unknown;
    const removed = Object.keys(prev).find((field) => !(field in cur));
    if (removed != null) throw new Error(`Cannot remove the field ${removed}`);
    return Object.fromEntries(
      Object.entries(cur).filter(([field, value]) => !deep.equal(prev[field], value)),
    );
  };
  const write = (values: unknown, edit: (copy: unknown) => void): Action =>
    action(changed(values, edited(values, edit)));
  return {
    set: (values, _, formPath, value) =>
      write(values, (copy) => deep.set(copy, formPath, value)),
    remove: (values, _, formPath) =>
      write(values, (copy) => deep.remove(copy, formPath)),
    insert: (values, path, value, after) =>
      write(values, (copy) =>
        patch.apply(copy as record.Unknown, {
          type: "insert",
          insert: { path, value, after, guards: [] },
        }),
      ),
    move: (values, path, key, after) =>
      write(values, (copy) =>
        patch.apply(copy as record.Unknown, {
          type: "move",
          move: { path, key, after, guards: [] },
        }),
      ),
    diff: (values, next) => {
      const fields = changed(values, next);
      return Object.keys(fields).length === 0 ? null : action(fields);
    },
  };
};

const overlaps = (a: string, b: string): boolean =>
  deep.pathsMatch(a, b) || deep.pathsMatch(b, a);

interface StoreParams<
  Key extends record.Key,
  Doc extends query.Data,
  Query extends DocumentQuery<Key>,
  V,
  Action,
> {
  client: Client;
  query: Query;
  domain: DocumentDomain<Key, Doc, Action>;
  schema: z.ZodType;
  toValues: (doc: Doc, query: Query) => V | undefined;
  codec: Codec<Action>;
  onError: (error: unknown) => void;
}

/** A form store that a document form holds for one document. */
interface Store<Query, V> extends Form.Store<V> {
  readonly client: Client;
  readonly query: Query;
  flush: () => Promise<boolean>;
  close: () => void;
}

/**
 * A form store over a document in the client cache. A write applies to the cached
 * document at once and reaches the Core as the action its codec builds. Writes to one
 * field coalesce into one dispatch, and a field whose value fails its schema holds its
 * edits until the value passes. A batch and a list edit send at once, and the Core
 * checks their values. A write of the whole form sends at once when the codec diffs,
 * and is a write to each field it changes otherwise. Sends run in order, so the Core
 * applies edits in the order they were made. An edit that another client made stale
 * drops quietly.
 */
class DocumentStore<
  Key extends record.Key,
  Doc extends query.Data,
  Query extends DocumentQuery<Key>,
  V,
  Action,
> implements Store<Query, V> {
  readonly client: Client;
  readonly query: Query;
  private readonly key: Key;
  private readonly params: StoreParams<Key, Doc, Query, V, Action>;
  private doc: Doc;
  private values: V;
  /** Whether the cached document no longer holds the edited part. */
  private removed = false;
  private readonly edits: Edit<Action>[] = [];
  private batched: actions.Transaction<Action> | null = null;
  private readonly inFlight = new Set<Promise<boolean>>();

  constructor(params: StoreParams<Key, Doc, Query, V, Action>) {
    this.params = params;
    this.client = params.client;
    this.query = params.query;
    this.key = params.query.key;
    const cached = params.domain.watch(this.key).getCached();
    if (!query.isLive(cached)) throw new Error(`Document ${this.key} is not cached`);
    const values = params.toValues(cached, this.query);
    if (values === undefined)
      throw new Error(`Document ${this.key} holds nothing for the form`);
    this.doc = cached;
    this.values = values;
  }

  get = (): V => {
    const cached = this.params.domain.watch(this.key).getCached();
    // A deleted document, or one whose edited part another client removed, keeps its
    // last values and takes no edits.
    if (query.isLive(cached) && cached !== this.doc) {
      this.doc = cached;
      const next = this.params.toValues(cached, this.query);
      this.removed = next === undefined;
      // Unchanged values keep their identity, so a change re-renders only the fields it
      // reaches, even when a create replaces the whole document.
      if (next !== undefined) this.values = deep.reuse(this.values, next);
    }
    return this.values;
  };

  set = (formPath: string, value: unknown): void => {
    if (formPath === "") {
      this.setChanged(value);
      return;
    }
    const values = this.get();
    const target = locate(this.params.schema, values, formPath);
    if (target == null) return;
    const { schema } = this.params;
    const field = zod.getFieldSchema(schema, formPath, { optional: true, values });
    const valid = field == null || field.safeParse(value).success;
    const whole = wholeAt(target, value);
    if (deep.equal(whole, target.node)) return;
    this.write(target.path, whole, valid, () =>
      this.params.codec.set(this.get(), target, formPath, value),
    );
  };

  remove = (formPath: string): void => {
    const values = this.get();
    const target = locate(this.params.schema, values, formPath);
    if (target == null || target.path.length === 0) return;
    this.dispatch(this.params.codec.remove(values, target, formPath));
  };

  insert = <E extends Form.Entry>(formPath: string, value: E, after?: string): void => {
    const values = this.get();
    const path = this.locateList(values, formPath);
    if (path == null) return;
    this.dispatch(this.params.codec.insert(values, path, value, after));
  };

  move = (formPath: string, key: string, after?: string): void => {
    const values = this.get();
    const path = this.locateList(values, formPath);
    if (path == null) return;
    this.dispatch(this.params.codec.move(values, path, key, after));
  };

  batch = (fn: () => void): void => {
    if (this.batched != null) {
      fn();
      return;
    }
    void this.takeValid();
    const tx = this.params.domain.beginTransaction(this.key);
    this.batched = tx;
    try {
      fn();
    } finally {
      this.batched = null;
      this.dropReplaced();
      void this.send(tx);
    }
  };

  onChange = (callback: () => void): destructor.Destructor =>
    this.params.domain.watch(this.key).onChange(() => callback());

  /**
   * Sends every edit the store holds, and waits for the sends already in flight.
   * @returns false when a field holds a value that fails its schema, which stays
   * unsent, or when one of those sends failed or went stale.
   */
  async flush(): Promise<boolean> {
    const sends = [...this.inFlight, ...this.takeValid()];
    const held = this.edits.length > 0;
    const sent = await Promise.all(sends);
    return !held && sent.every(Boolean);
  }

  /** Sends every valid edit and drops every invalid one. The store stays usable. */
  close(): void {
    void this.takeValid();
    this.edits.splice(0).forEach((edit) => edit.tx.abort());
  }

  private setChanged(next: unknown): void {
    const values = this.get();
    const { codec, schema } = this.params;
    if (codec.diff != null) {
      const action = codec.diff(values, next);
      if (action != null) this.dispatch(action);
      return;
    }
    patch.diff(schema, values, next).forEach((action) => {
      const join = (path: string[]) => path.join(deep.SEPARATOR);
      switch (action.type) {
        case "set": {
          const { path, value } = action.set;
          // A change of union member at the root has no field to write.
          if (path.length > 0) this.set(join(path), value);
          else {
            const root = { path, node: values, rest: "", entry: false };
            this.dispatch(codec.set(values, root, "", value));
          }
          return;
        }
        case "remove":
          this.remove(join([...action.remove.path, action.remove.key]));
          return;
        case "insert": {
          const { path, value, after } = action.insert;
          this.insert(join(path), value as Form.Entry, after);
          return;
        }
        case "move": {
          const { path, key, after } = action.move;
          this.move(join(path), key, after);
        }
      }
    });
  }

  private locateList(values: V, formPath: string): string[] | null {
    const target = locate(this.params.schema, values, formPath);
    if (target == null) return null;
    if (target.rest !== "")
      throw new Error(`${formPath} is inside a list of entries without keys`);
    return target.path;
  }

  /**
   * Writes the value at path through the action that build returns. Build runs after
   * the write drops the earlier writes that it replaces.
   */
  private write(
    path: string[],
    value: unknown,
    valid: boolean,
    build: () => Action,
  ): void {
    if (this.removed) return;
    if (this.batched != null) {
      this.batched.add(build());
      return;
    }
    const at = path.join(deep.SEPARATOR);
    let edit = this.edits.find((e) =>
      [...e.writes.keys()].some((p) => overlaps(p, at)),
    );
    if (edit == null) {
      edit = { tx: this.params.domain.beginTransaction(this.key), writes: new Map() };
      this.edits.push(edit);
    }
    // A write replaces every earlier write below its path.
    const replaced = [...edit.writes.keys()].filter((p) => deep.pathsMatch(p, at));
    if (replaced.length > 0) {
      // The Core applies each action of a dispatch, so a replaced value that fails its
      // schema would fail the dispatch.
      for (const p of replaced) edit.writes.delete(p);
      edit.tx.abort();
      edit.tx = this.params.domain.beginTransaction(this.key);
      edit.tx.add([...edit.writes.values()].map(({ action }) => action));
    }
    const action = build();
    edit.writes.set(at, { action, value, valid });
    edit.tx.add(action);
    clearTimeout(edit.timer);
    const scheduled = edit;
    edit.timer = setTimeout(
      () => void this.take(scheduled),
      COALESCE_WINDOW.milliseconds,
    );
  }

  /** Sends the action right after the edits before it. */
  private dispatch(action: Action): void {
    if (this.removed) return;
    if (this.batched != null) {
      this.batched.add(action);
      return;
    }
    void this.takeValid();
    const tx = this.params.domain.beginTransaction(this.key);
    tx.add(action);
    this.dropReplaced();
    void this.send(tx);
  }

  /** Takes every valid edit out of the store and queues its send. */
  private takeValid(): Promise<boolean>[] {
    const sends = [...this.edits].flatMap((edit) => this.take(edit) ?? []);
    this.dropReplaced();
    return sends;
  }

  /**
   * Aborts each held edit whose value the document no longer shows, as when a batch
   * replaced the value or another client removed its entry. Such an edit has nothing
   * left to correct, and it would write over the newer value once it sends.
   */
  private dropReplaced(): void {
    const values = this.get();
    for (const edit of [...this.edits]) {
      const shown = [...edit.writes].some(
        ([at, { value, valid }]) =>
          !valid && deep.equal(deep.get(values, at, { optional: true }), value),
      );
      if (shown && !this.removed) continue;
      clearTimeout(edit.timer);
      this.edits.splice(this.edits.indexOf(edit), 1);
      edit.tx.abort();
    }
  }

  /**
   * Takes the edit out of the store and queues its send, unless it holds a value that
   * fails its schema.
   * @returns the queued send, or null when the edit stays.
   */
  private take(edit: Edit<Action>): Promise<boolean> | null {
    clearTimeout(edit.timer);
    if (![...edit.writes.values()].every(({ valid }) => valid)) return null;
    this.edits.splice(this.edits.indexOf(edit), 1);
    return this.send(edit.tx);
  }

  /**
   * @returns false when the send failed or went stale. A transaction with nothing to
   * send passes.
   */
  private async send(tx: actions.Transaction<Action>): Promise<boolean> {
    const sent = this.commit(tx);
    this.inFlight.add(sent);
    try {
      return await sent;
    } finally {
      this.inFlight.delete(sent);
    }
  }

  private async commit(tx: actions.Transaction<Action>): Promise<boolean> {
    try {
      await tx.commit();
      return true;
    } catch (e) {
      // The cached document drops a stale edit, and the form shows the document.
      if (!patch.StaleError.matches(e)) this.params.onError(e);
      return false;
    }
  }
}

interface DocumentFormBase<
  Key extends record.Key,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key>,
  Doc,
> {
  name: string;
  /**
   * The form's schema. A field write outside a batch waits until its value passes its
   * field schema.
   */
  schema: Schema;
  /** Suspends until the document is cached. */
  useEnsure: (query: DocumentQuery<Key>) => void;
  /**
   * Projects the edited part of the document onto the form's values. Each field keeps
   * its path in that part.
   * @returns undefined when the document no longer holds the part, as when another
   * client removed it. The form then keeps its last values and takes no edits.
   */
  toValues: (doc: Doc, query: Query) => z.infer<Schema> | undefined;
}

/**
 * Sends each edit as a path action. The discriminated unions of the form's schema on a
 * path decide the guards of its action.
 */
interface PathDocumentFormParams<
  Key extends record.Key,
  Doc extends query.Data,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key>,
> extends DocumentFormBase<Key, Schema, Query, Doc> {
  domain: (client: Client) => DocumentDomain<Key, Doc>;
  /**
   * The path of the edited part in the document, such as the key of one entry in a map.
   * The form edits the whole document when omitted.
   */
  pathOf?: (query: Query) => string[];
  updates?: never;
  fields?: never;
}

/**
 * Sends each edit as an update of the form's values. A write of the whole form sends
 * the update that turns the form's values into the written ones.
 */
interface UpdateDocumentFormParams<
  Key extends record.Key,
  Doc extends query.Data,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key>,
  Action,
  Update extends z.ZodType,
> extends DocumentFormBase<Key, Schema, Query, Doc> {
  domain: (client: Client) => DocumentDomain<Key, Doc, Action>;
  updates: DocumentUpdates<Action, Update>;
  pathOf?: never;
  fields?: never;
}

/**
 * Sends each edit as a named action that sets the top-level fields of the form's
 * values that the edit changes. Such an action cannot remove a top-level field.
 */
interface FieldsDocumentFormParams<
  Key extends record.Key,
  Doc extends query.Data,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key>,
  Action,
> extends DocumentFormBase<Key, Schema, Query, Doc> {
  domain: (client: Client) => DocumentDomain<Key, Doc, Action>;
  /** Builds the action that sets the fields. */
  fields: (query: Query, fields: record.Unknown) => Action;
  pathOf?: never;
  updates?: never;
}

export type CreateDocumentFormParams<
  Key extends record.Key,
  Doc extends query.Data,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key>,
  Action = patch.Action,
  Update extends z.ZodType = z.ZodType,
> =
  | PathDocumentFormParams<Key, Doc, Schema, Query>
  | UpdateDocumentFormParams<Key, Doc, Schema, Query, Action, Update>
  | FieldsDocumentFormParams<Key, Doc, Schema, Query, Action>;

export interface UseDocumentFormParams<Query> extends Pick<
  Form.UseParams<z.ZodType>,
  "mode" | "onHasTouched"
> {
  query: Query;
}

export interface UseDocumentFormReturn<Schema extends z.ZodType> {
  form: Form.UseReturn<Schema>;
  /**
   * Sends every edit the form holds.
   * @returns false when a field holds a value that fails its schema, or a send failed
   * or went stale.
   */
  flush: () => Promise<boolean>;
}

export interface UseDocumentForm<Query, Schema extends z.ZodType> {
  (params: UseDocumentFormParams<Query>): UseDocumentFormReturn<Schema>;
}

const WHOLE_DOCUMENT = (): string[] => [];

/**
 * Builds a hook for a form over a document in the client cache, which other clients
 * edit at the same time. The form shows the cached document, re-rendering only the
 * fields a remote edit reaches. Each edit reaches the Core as an update, a named
 * action, or a path action, as the params choose. Call it once per document type, at
 * module scope.
 */
export const createDocumentForm = <
  Key extends record.Key,
  Doc extends query.Data,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key> = DocumentQuery<Key>,
  Action = patch.Action,
  Update extends z.ZodType = z.ZodType,
>(
  params: CreateDocumentFormParams<Key, Doc, Schema, Query, Action, Update>,
): UseDocumentForm<Query, Schema> => {
  const { name, schema, useEnsure, toValues } = params;
  type V = z.infer<Schema>;
  const createStore = (
    client: Client,
    query: Query,
    onError: (error: unknown) => void,
  ): Store<Query, V> => {
    const base = { client, query, schema, toValues, onError };
    if (params.updates != null)
      return new DocumentStore({
        ...base,
        domain: params.domain(client),
        codec: updateCodec(params.updates),
      });
    if (params.fields != null) {
      const { fields } = params;
      return new DocumentStore({
        ...base,
        domain: params.domain(client),
        codec: fieldsCodec((changed) => fields(query, changed)),
      });
    }
    const { pathOf = WHOLE_DOCUMENT } = params;
    return new DocumentStore({
      ...base,
      domain: params.domain(client),
      codec: pathCodec(schema, pathOf(query)),
    });
  };
  const useDocumentForm = ({
    query,
    mode,
    onHasTouched,
  }: UseDocumentFormParams<Query>): UseDocumentFormReturn<Schema> => {
    useEnsure({ key: query.key });
    const client = Synnax.use();
    const handleError = Status.useErrorHandler();
    const create = (): Store<Query, V> => {
      if (client == null)
        throw new DisconnectedError(`Cannot edit ${name}: no Core connected.`);
      return createStore(client, query, (error) =>
        handleError(error, `Failed to save ${name}`),
      );
    };
    const [store, setStore] = useState(create);
    if (store.client !== client || !deep.equal(store.query, query)) setStore(create());
    useEffect(() => () => store.close(), [store]);
    const form = Form.use<Schema>({ store, schema, mode, onHasTouched });
    const flush = async (): Promise<boolean> => await store.flush();
    return { form, flush };
  };
  return useDocumentForm;
};
