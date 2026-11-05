// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ontology,
  query,
  resource,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { type destructor, type record } from "@synnaxlabs/x";

/**
 * A live list holding some of the children of a tree node. It reads each listed
 * resource by key as well, so a change to one resource reaches only its own row.
 */
export interface Source {
  /** Fetches the list. The answer reaches onChange handlers too. */
  retrieve: () => Promise<ontology.Resource[]>;
  /** Calls the handler with the list each time it changes. */
  onChange: (
    handler: (resources: ontology.Resource[]) => void,
  ) => destructor.Destructor;
  /** Returns the cached list, or undefined when nothing is cached. */
  getCached: () => ontology.Resource[] | undefined;
  /** Returns the cached resource under the key of a resource the list holds. */
  get: (key: string) => ontology.Resource | undefined;
  /** Calls the handler each time the resource under the key changes. */
  subscribe: (key: string, handler: () => void) => destructor.Destructor;
}

export interface ChildrenProps {
  client: Client;
  /** The node to list the children of. */
  id: resource.ID;
  /** The root of the tree. Its type names the scope of the tree. */
  root: resource.ID;
}

/** Returns the sources that together list the children of a node. */
export interface Children {
  (props: ChildrenProps): Source[];
}

/** A typed client that answers list queries and reads records by key, both live. */
export interface Retriever<Q, D extends record.Keyed<record.Key>> {
  retrieve: (query: Q) => Promise<D[]>;
  onChange: {
    (key: D["key"], handler: query.ChangeHandler<D>): destructor.Destructor;
    (query: Q, handler: query.ChangeHandler<D[]>): destructor.Destructor;
  };
  getCached: {
    (key: D["key"]): query.Cached<D> | undefined;
    (query: Q): query.Cached<D[]> | undefined;
  };
}

interface Reads<D extends record.Keyed<record.Key>> {
  retrieve: () => Promise<D[]>;
  onList: (handler: query.ChangeHandler<D[]>) => destructor.Destructor;
  getList: () => query.Cached<D[]> | undefined;
  onRecord: (key: D["key"], handler: query.ChangeHandler<D>) => destructor.Destructor;
  getRecord: (key: D["key"]) => query.Cached<D> | undefined;
}

const fromReads = <D extends record.Keyed<record.Key>>(
  reads: Reads<D>,
  toResource: (record: D) => ontology.Resource,
  filter: (record: D) => boolean,
): Source => {
  // A record keeps its identity until it changes, so a row reads the same resource
  // until then.
  const resources = new WeakMap<D, ontology.Resource>();
  const keys = new Map<string, D["key"]>();
  const convert = (record: D): ontology.Resource => {
    let res = resources.get(record);
    if (res == null) {
      res = toResource(record);
      resources.set(record, res);
    }
    return res;
  };
  const convertList = (records: D[]): ontology.Resource[] => {
    const listed = records.filter(filter);
    keys.clear();
    return listed.map((record) => {
      const res = convert(record);
      keys.set(resource.idToString(res.id), record.key);
      return res;
    });
  };
  return {
    retrieve: async () => convertList(await reads.retrieve()),
    onChange: (handler) =>
      reads.onList((answer) => {
        if (query.isLive(answer)) handler(convertList(answer));
      }),
    getCached: () => {
      const cached = reads.getList();
      return query.isLive(cached) ? convertList(cached) : undefined;
    },
    get: (key) => {
      const recordKey = keys.get(key);
      if (recordKey == null) return undefined;
      const cached = reads.getRecord(recordKey);
      return query.isLive(cached) ? convert(cached) : undefined;
    },
    subscribe: (key, handler) => {
      const recordKey = keys.get(key);
      if (recordKey == null) return () => {};
      return reads.onRecord(recordKey, () => handler());
    },
  };
};

/**
 * Builds a source that lists the records a typed client answers for the query, keeping
 * only the records that pass the filter.
 */
export const createSource = <Q, D extends record.Keyed<record.Key>>(
  retriever: Retriever<Q, D>,
  q: Q,
  toResource: (record: D) => ontology.Resource,
  filter: (record: D) => boolean = () => true,
): Source =>
  fromReads(
    {
      retrieve: async () => await retriever.retrieve(q),
      onList: (handler) => retriever.onChange(q, handler),
      getList: () => retriever.getCached(q),
      onRecord: (key, handler) => retriever.onChange(key, handler),
      getRecord: (key) => retriever.getCached(key),
    },
    toResource,
    filter,
  );

/** Returns a function that shows a typed record of the type as a tree resource. */
export const toResource =
  (type: resource.Type) =>
  <D extends record.KeyedNamed<record.Key>>(record: D): ontology.Resource => {
    const id: resource.ID = { type, key: String(record.key) };
    return {
      key: resource.idToString(id),
      id,
      name: record.name,
      data: record as unknown as record.Unknown,
    };
  };

/** Builds a source that lists the ontology children of the node of the given types. */
export const createOntologySource = (
  client: Client,
  id: resource.ID,
  types?: resource.Type[],
  filter: (resource: ontology.Resource) => boolean = () => true,
): Source => {
  const q = { ids: id, types };
  const { ontology: resources } = client;
  return fromReads(
    {
      retrieve: async () => await resources.children.retrieve(q),
      onList: (handler) => resources.children.onChange(q, handler),
      getList: () => resources.children.getCached(q),
      onRecord: (key, handler) => resources.onChange(key, handler),
      getRecord: (key) => resources.getCached(key),
    },
    (r) => r,
    filter,
  );
};
