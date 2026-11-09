// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type query, resource, type Synnax as Client } from "@synnaxlabs/client";
import { type destructor, type record } from "@synnaxlabs/x";

import { type Resource } from "@/platform/tree/types";

/** What a child list of a tree node holds now. */
export interface Window {
  /** The listed resources, in name order. */
  resources: Resource[];
  /** Whether more resources match past the last one listed. */
  more: boolean;
}

/** A live list holding some of the children of a tree node, in name order. */
export interface Source {
  /** Fetches the list. The window reaches onChange handlers too. */
  retrieve: () => Promise<Window>;
  /** Fetches the next page of the list. The window reaches onChange handlers too. */
  more: () => Promise<Window>;
  /** Returns the window, or undefined before the list is fetched. */
  getCached: () => Window | undefined;
  /**
   * Calls the handler each time the window changes, with undefined once the window is
   * discarded.
   */
  onChange: (handler: (window: Window | undefined) => void) => destructor.Destructor;
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

interface NameOrder {
  order: { field: "name" };
}

const NAME_ORDER: NameOrder["order"] = { field: "name" };

/** The most children that a list fetches at a time. */
const PAGE_SIZE = 1000;

/** A typed client that keeps live lists of the records that queries match. */
export interface Retriever<Q, D extends record.Keyed<record.Key>> {
  watch: (query: Q & NameOrder & { limit: number }) => query.List<D, unknown>;
}

/** Builds a source that lists the records a typed client matches for the query. */
export const createSource = <Q, D extends record.Keyed<record.Key>>(
  retriever: Retriever<Q, D>,
  q: Q,
  toResource: (record: D) => Resource,
): Source => {
  const list = retriever.watch({ ...q, order: NAME_ORDER, limit: PAGE_SIZE });
  // A record keeps its identity until it changes, so a row reads the same resource
  // until then.
  const resources = new WeakMap<D, Resource>();
  const convert = (record: D): Resource => {
    let res = resources.get(record);
    if (res == null) {
      res = toResource(record);
      resources.set(record, res);
    }
    return res;
  };
  let held: { from: query.Window<D, unknown>; window: Window } | undefined;
  const toWindow = (from: query.Window<D, unknown>): Window => {
    if (held?.from !== from)
      held = {
        from,
        window: { resources: from.records.map(convert), more: from.end != null },
      };
    return held.window;
  };
  return {
    retrieve: async () => toWindow(await list.retrieve()),
    more: async () => toWindow(await list.more(PAGE_SIZE)),
    getCached: () => {
      const cached = list.getCached();
      return cached == null ? undefined : toWindow(cached);
    },
    onChange: (handler) =>
      list.onChange((window) => handler(window == null ? undefined : toWindow(window))),
  };
};

/** Returns a function that shows a typed record of the type as a tree resource. */
export const toResource =
  (type: resource.Type) =>
  <D extends record.KeyedNamed<record.Key>>(record: D): Resource => {
    const id: resource.ID = { type, key: String(record.key) };
    return {
      key: resource.idToString(id),
      id,
      name: record.name,
      data: record as unknown as record.Unknown,
    };
  };
