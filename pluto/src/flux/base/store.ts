// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache, type dispatch, type Synnax } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import type z from "zod";

// Store mechanism lives in the client cache package; flux re-exports the
// surface so pluto consumers keep one namespace.

export type UnaryStore<
  Key extends record.Key = record.Key,
  Value extends cache.State = cache.State,
  SetExtra extends unknown | undefined = undefined,
> = cache.UnaryStore<Key, Value, SetExtra>;

/**
 * Base interface for a collection of UnaryStore instances.
 * Each property is a UnaryStore with its own key-value type.
 */
export interface Store extends cache.Stores {}

export type ChannelListener<
  ScopedStore extends Store = {},
  Z extends z.ZodType = z.ZodType,
> = cache.ChannelListener<ScopedStore, Z>;

export type ChannelListenerParams<
  ScopedStore extends Store = {},
  Z extends z.ZodType = z.ZodType,
> = cache.ChannelListenerParams<ScopedStore, Z>;

export type UnaryStoreConfig<
  ScopedStore extends Store = {},
  Key extends record.Key = record.Key,
  Value extends cache.State = cache.State,
> = cache.UnaryStoreConfig<ScopedStore, Key, Value>;

export type StoreConfig<ScopedStore extends Store = {}> =
  cache.StoreConfig<ScopedStore>;

export type SetHandler<
  Value,
  SetExtra extends unknown | undefined = undefined,
> = cache.SetHandler<Value, SetExtra>;

export type DeleteHandler<K extends record.Key> = cache.DeleteHandler<K>;

export type QueryState<D extends cache.Data> = cache.QueryState<D>;

export const orderByKeys = cache.orderByKeys;
export const partialUpdate = cache.partialUpdate;
export const hashQuery = cache.hashQuery;

export interface StoreComposerParams {
  /** Non-null only when the engine belongs to a connected client. */
  client: Synnax | null;
  engine: cache.Engine;
}

/** Builds the dispatch controller backing an undoable store key. */
export interface StoreComposer {
  (params: StoreComposerParams): dispatch.Controller<any, any, any>;
}

/** Controller composers keyed by the store key they overlay. */
export interface StoreComposers {
  [storeKey: string]: StoreComposer;
}

/**
 * Lazily materializes scoped store handles from the engine, overlaying the
 * dispatch handle for keys with a controller. Handles are cached per key so
 * repeated property reads return the same instance.
 */
export const createScopedStore = <ScopedStore extends Store>(
  engine: cache.Engine,
  controllers: Record<string, dispatch.Controller<any, any, any>>,
  scope: string,
): ScopedStore => {
  const handles = new Map<PropertyKey, unknown>();
  return new Proxy({} as ScopedStore, {
    get: (_, key) => {
      // Promise resolution probes .then on returned values; never a store key.
      if (key === "then") return undefined;
      let handle = handles.get(key);
      if (handle == null && typeof key === "string") {
        const store = engine.store(key, scope);
        const controller = controllers[key];
        handle = controller == null ? store : { ...store, ...controller.scope(scope) };
        handles.set(key, handle);
      }
      return handle;
    },
  });
};
