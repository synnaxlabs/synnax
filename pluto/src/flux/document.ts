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
export interface DocumentDomain<Key extends record.Key, Doc extends query.Data> {
  getCached: (key: Key) => query.Cached<Doc> | undefined;
  onChange: (key: Key, handler: query.ChangeHandler<Doc>) => destructor.Destructor;
  /** Begins a transaction of actions on the cached document, with camelCase keys. */
  beginTransaction: (key: Key) => actions.Transaction<patch.Action>;
}

/** Names the document that a document form edits. */
export interface DocumentQuery<Key extends record.Key> {
  key: Key;
}

/** How long a field holds its edits before it sends them as one dispatch. */
const COALESCE_WINDOW = TimeSpan.milliseconds(200);

/** The edits to one field that have not been sent. */
interface Edit {
  tx: actions.Transaction<patch.Action>;
  /** Whether the value at each path that the edit writes passes its field schema. */
  valid: Map<string, boolean>;
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
  let i = 0;
  while (i < parts.length) {
    if (Array.isArray(node)) {
      const entry = deep.defaultGetter(node as unknown as record.Unknown, parts[i]);
      if (isEntry(entry)) {
        path.push(entry.key);
        node = entry;
        i++;
        continue;
      }
      if (isKeyedList(schema, parts.slice(0, i).join(deep.SEPARATOR), values))
        return null;
      return { path, node, rest: parts.slice(i).join(deep.SEPARATOR) };
    }
    if (node == null || typeof node !== "object") break;
    const best = deep.findBestKey(node as record.Unknown, parts.slice(i));
    if (best == null) break;
    const [key, consumed] = best;
    path.push(key);
    node = (node as record.Unknown)[key];
    i += consumed;
  }
  if (i < parts.length) {
    path.push(...parts.slice(i));
    node = undefined;
  }
  return { path, node, rest: "" };
};

const overlaps = (a: string, b: string): boolean =>
  deep.pathsMatch(a, b) || deep.pathsMatch(b, a);

interface StoreParams<
  Key extends record.Key,
  Doc extends query.Data,
  Query extends DocumentQuery<Key>,
  V,
> {
  client: Client;
  query: Query;
  domain: DocumentDomain<Key, Doc>;
  schema: z.ZodType;
  toValues: (doc: Doc, query: Query) => V | undefined;
  /** The path of the edited part in the document. */
  path: string[];
  onError: (error: unknown) => void;
}

/**
 * A form store over a document in the client cache. A write applies to the cached
 * document at once and reaches the Core as a path action. Writes to one field coalesce
 * into one dispatch, and a field whose value fails its schema holds its edits until the
 * value passes. A batch and a list edit send at once, and the Core checks their values.
 * Sends run in order, so the Core applies edits in the order they were made.
 */
class DocumentStore<
  Key extends record.Key,
  Doc extends query.Data,
  Query extends DocumentQuery<Key>,
  V,
> implements Form.Store<V> {
  readonly client: Client;
  readonly query: Query;
  private readonly key: Key;
  private readonly params: StoreParams<Key, Doc, Query, V>;
  private doc: Doc;
  private values: V;
  /** Whether the cached document no longer holds the edited part. */
  private removed = false;
  private readonly edits: Edit[] = [];
  private batched: actions.Transaction<patch.Action> | null = null;
  private readonly inFlight = new Set<Promise<boolean>>();

  constructor(params: StoreParams<Key, Doc, Query, V>) {
    this.params = params;
    this.client = params.client;
    this.query = params.query;
    this.key = params.query.key;
    const cached = params.domain.getCached(this.key);
    if (!query.isLive(cached)) throw new Error(`Document ${this.key} is not cached`);
    const values = params.toValues(cached, this.query);
    if (values === undefined)
      throw new Error(`Document ${this.key} holds nothing at ${params.path.join(".")}`);
    this.doc = cached;
    this.values = values;
  }

  get = (): V => {
    const cached = this.params.domain.getCached(this.key);
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
    if (formPath === "") throw new Error("A document form cannot replace its document");
    const values = this.get();
    const target = locate(this.params.schema, values, formPath);
    if (target == null) return;
    const { schema } = this.params;
    const field = zod.getFieldSchema(schema, formPath, { optional: true, values });
    const valid = field == null || field.safeParse(value).success;
    let whole = value;
    if (target.rest !== "") {
      whole = deep.copy(target.node);
      deep.set(whole, target.rest, value);
    }
    if (deep.equal(whole, target.node)) return;
    const guards = patch.guards(this.params.schema, target.path, values);
    this.write(
      target.path.join(deep.SEPARATOR),
      { type: "set", set: { path: target.path, value: whole, guards } },
      valid,
    );
  };

  remove = (formPath: string): void => {
    const values = this.get();
    const target = locate(this.params.schema, values, formPath);
    if (target == null || target.path.length === 0) return;
    const guards = patch.guards(this.params.schema, target.path, values);
    if (target.rest !== "") {
      const whole = deep.copy(target.node);
      deep.remove(whole, target.rest);
      this.dispatch({ type: "set", set: { path: target.path, value: whole, guards } });
      return;
    }
    const path = target.path.slice(0, -1);
    const key = target.path[target.path.length - 1];
    this.dispatch({ type: "remove", remove: { path, key, guards } });
  };

  insert = <E extends Form.Entry>(formPath: string, value: E, after?: string): void => {
    const values = this.get();
    const path = this.locateList(values, formPath);
    if (path == null) return;
    const guards = patch.guards(this.params.schema, path, values);
    this.dispatch({ type: "insert", insert: { path, value, after, guards } });
  };

  move = (formPath: string, key: string, after?: string): void => {
    const values = this.get();
    const path = this.locateList(values, formPath);
    if (path == null) return;
    const guards = patch.guards(this.params.schema, path, values);
    this.dispatch({ type: "move", move: { path, key, after, guards } });
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
      void this.send(tx);
    }
  };

  onChange = (callback: () => void): destructor.Destructor =>
    this.params.domain.onChange(this.key, () => callback());

  /**
   * Sends every edit the store holds, and waits for the sends already in flight.
   * @returns false when a field holds a value that fails its schema, which stays
   * unsent, or when one of those sends failed.
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

  private locateList(values: V, formPath: string): string[] | null {
    const target = locate(this.params.schema, values, formPath);
    if (target == null) return null;
    if (target.rest !== "")
      throw new Error(`${formPath} is inside a list of entries without keys`);
    return target.path;
  }

  private write(at: string, action: patch.Action, valid: boolean): void {
    if (this.removed) return;
    const prefixed = patch.prefix(action, this.params.path);
    if (this.batched != null) {
      this.batched.add(prefixed);
      return;
    }
    let edit = this.edits.find((e) => [...e.valid.keys()].some((p) => overlaps(p, at)));
    if (edit == null) {
      edit = { tx: this.params.domain.beginTransaction(this.key), valid: new Map() };
      this.edits.push(edit);
    }
    edit.tx.add(prefixed);
    // A write replaces every earlier write below its path.
    for (const p of [...edit.valid.keys()])
      if (deep.pathsMatch(p, at)) edit.valid.delete(p);
    edit.valid.set(at, valid);
    clearTimeout(edit.timer);
    const scheduled = edit;
    edit.timer = setTimeout(
      () => void this.take(scheduled),
      COALESCE_WINDOW.milliseconds,
    );
  }

  /** Sends the field's edits right after the edits before them. */
  private dispatch(action: patch.Action): void {
    if (this.removed) return;
    const prefixed = patch.prefix(action, this.params.path);
    if (this.batched != null) {
      this.batched.add(prefixed);
      return;
    }
    void this.takeValid();
    const tx = this.params.domain.beginTransaction(this.key);
    tx.add(prefixed);
    void this.send(tx);
  }

  /** Takes every valid edit out of the store and queues its send. */
  private takeValid(): Promise<boolean>[] {
    return [...this.edits].flatMap((edit) => this.take(edit) ?? []);
  }

  /**
   * Takes the edit out of the store and queues its send, unless it holds a value that
   * fails its schema.
   * @returns the queued send, or null when the edit stays.
   */
  private take(edit: Edit): Promise<boolean> | null {
    clearTimeout(edit.timer);
    if (![...edit.valid.values()].every(Boolean)) return null;
    this.edits.splice(this.edits.indexOf(edit), 1);
    return this.send(edit.tx);
  }

  /** @returns false when the send failed. A transaction with nothing to send passes. */
  private async send(tx: actions.Transaction<patch.Action>): Promise<boolean> {
    const sent = this.commit(tx);
    this.inFlight.add(sent);
    try {
      return await sent;
    } finally {
      this.inFlight.delete(sent);
    }
  }

  private async commit(tx: actions.Transaction<patch.Action>): Promise<boolean> {
    try {
      await tx.commit();
      return true;
    } catch (e) {
      this.params.onError(e);
      return false;
    }
  }
}

export interface CreateDocumentFormParams<
  Key extends record.Key,
  Doc extends query.Data,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key>,
> {
  name: string;
  /**
   * The form's schema. A field write outside a batch waits until its value passes its
   * field schema, and the discriminated unions of a path decide the guards of its
   * action.
   */
  schema: Schema;
  domain: (client: Client) => DocumentDomain<Key, Doc>;
  /** Suspends until the document is cached. */
  useEnsure: (query: DocumentQuery<Key>) => void;
  /**
   * Projects the edited part of the document onto the form's values. Each field keeps
   * its path in that part.
   * @returns undefined when the document no longer holds the part, as when another
   * client removed it. The form then keeps its last values and takes no edits.
   */
  toValues: (doc: Doc, query: Query) => z.infer<Schema> | undefined;
  /**
   * The path of the edited part in the document, such as the key of one entry in a map.
   * The form edits the whole document when omitted.
   */
  pathOf?: (query: Query) => string[];
}

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
   * @returns false when a field holds a value that fails its schema, or a send failed.
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
 * fields a remote edit reaches, and each edit reaches the Core as a path action. Call
 * it once per document type, at module scope.
 */
export const createDocumentForm = <
  Key extends record.Key,
  Doc extends query.Data,
  Schema extends z.ZodType,
  Query extends DocumentQuery<Key> = DocumentQuery<Key>,
>({
  name,
  schema,
  domain,
  useEnsure,
  toValues,
  pathOf = WHOLE_DOCUMENT,
}: CreateDocumentFormParams<Key, Doc, Schema, Query>): UseDocumentForm<
  Query,
  Schema
> => {
  const useDocumentForm = ({
    query,
    mode,
    onHasTouched,
  }: UseDocumentFormParams<Query>): UseDocumentFormReturn<Schema> => {
    useEnsure({ key: query.key });
    const client = Synnax.use();
    const handleError = Status.useErrorHandler();
    const create = (): DocumentStore<Key, Doc, Query, z.infer<Schema>> => {
      if (client == null)
        throw new DisconnectedError(`Cannot edit ${name}: no Core connected.`);
      return new DocumentStore({
        client,
        query,
        domain: domain(client),
        schema,
        toValues,
        path: pathOf(query),
        onError: (error) => handleError(error, `Failed to save ${name}`),
      });
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
