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

/** A live list holding some of the children of a tree node. */
export interface Source {
  /** Fetches the list. */
  retrieve: () => Promise<ontology.Resource[]>;
  /** Calls the handler with the list each time it changes. */
  onChange: (
    handler: (resources: ontology.Resource[]) => void,
  ) => destructor.Destructor;
  /** Returns the cached list, or undefined when nothing is cached. */
  getCached: () => ontology.Resource[] | undefined;
}

export interface ChildrenProps {
  client: Client;
  /** The node to list the children of. */
  id: resource.ID;
  /** The resource of the node, absent when the node is the tree root. */
  resource?: ontology.Resource;
}

/** Returns the sources that together list the children of a node. */
export interface Children {
  (props: ChildrenProps): Source[];
}

/** A typed client that answers a list query and keeps it live. */
export interface Retriever<Q, D> {
  retrieve: (query: Q) => Promise<D[]>;
  onChange: (query: Q, handler: query.ChangeHandler<D[]>) => destructor.Destructor;
  getCached: (query: Q) => query.Cached<D[]> | undefined;
}

/**
 * Builds a source that lists the records a typed client answers for the query, keeping
 * only the records that pass the filter.
 */
export const createSource = <Q, D>(
  retriever: Retriever<Q, D>,
  q: Q,
  toResource: (record: D) => ontology.Resource,
  filter: (record: D) => boolean = () => true,
): Source => {
  const convert = (records: D[]): ontology.Resource[] =>
    records.filter(filter).map(toResource);
  return {
    retrieve: async () => convert(await retriever.retrieve(q)),
    onChange: (handler) =>
      retriever.onChange(q, (answer) => {
        if (query.isLive(answer)) handler(convert(answer));
      }),
    getCached: () => {
      const cached = retriever.getCached(q);
      return query.isLive(cached) ? convert(cached) : undefined;
    },
  };
};

/** A typed record a tree can show. */
export interface Named extends record.Keyed<record.Key> {
  name: string;
}

/** Returns a function that shows a typed record of the type as a tree resource. */
export const toResource =
  (type: resource.Type) =>
  <D extends Named>(record: D): ontology.Resource => {
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
  filter?: (resource: ontology.Resource) => boolean,
): Source =>
  createSource(
    client.ontology.children,
    { ids: id, types },
    (r: ontology.Resource) => r,
    filter,
  );
