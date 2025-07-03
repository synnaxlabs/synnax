// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DisconnectedError, type Synnax } from "@synnaxlabs/client";
import {
  type Destructor,
  type MultiSeries,
  type primitive,
  status,
} from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { Sync } from "@/flux/sync";
import { state } from "@/state";

/**
 * Parameters used to retrieve and or/update a resource from within a query. The query
 * re-executes whenever the parameters change.
 */
export type Params = Record<string, primitive.Value>;

interface ListenerExtraArgs<QueryParams extends Params, Data extends state.State> {
  params: QueryParams;
  client: Synnax;
  onChange: state.Setter<Data>;
}

/**
 * Configuration for a listener that is called whenever a new value is received
 * from the specified channel. The listener is called with the new value and can
 * choose to update the state of the query by calling the `onChange` function.
 *
 * The listener will not be called if the query is in a loading or an error state.
 */
export interface ListenerConfig<QueryParams extends Params, Data extends state.State> {
  /** The channel to listen to. */
  channel: channel.Name;
  /** The function to call when a new value is received. */
  onChange: Sync.ListenerHandler<MultiSeries, ListenerExtraArgs<QueryParams, Data>>;
}

/**
 * Arguments passed to the `retrieve` function on the query.
 */
export interface RetrieveArgs<QueryParams extends Params> {
  client: Synnax;
  params: QueryParams;
}

/**
 * The result of a query. The query can be in one of three states:
 * - `loading` - Data is currently being fetched.
 * - `error` - An error occurred while fetching data, or while handling values provided
 * to a listener.
 * - `success` - Data was successfully fetched and is available in the `data` field.
 */
export type Result<Data extends state.State> =
  | (status.Status<undefined, "loading"> & {
      data: null;
      error: null;
    })
  | (status.Status<status.ExceptionDetails, "error"> & {
      data: null;
      error: unknown;
    })
  | (status.Status<undefined, "success"> & {
      data: Data;
      error: null;
    });

/** A factory function to create a loading result. */
export const loadingResult = <Data extends state.State>(
  name: string,
): Result<Data> => ({
  ...status.create<undefined, "loading">({
    variant: "loading",
    message: `Loading ${name}`,
  }),
  data: null,
  error: null,
});

/** A factory function to create a success result. */
export const successResult = <Data extends state.State>(
  name: string,
  value: Data,
): Result<Data> => ({
  ...status.create<undefined, "success">({
    variant: "success",
    message: `Loaded ${name}`,
  }),
  data: value,
  error: null,
});

/** A factory function to create an error result. */
export const errorResult = <Data extends state.State>(
  name: string,
  error: unknown,
): Result<Data> => ({
  ...status.fromException(error, `Failed to load ${name}`),
  data: null,
  error,
});

/**
 * Arguments passed to the `useBase` hook.
 * @template QParams - The type of the parameters for the query.
 * @template Data - The type of the data being retrieved.
 */
export interface UseObservableArgs<QParams extends Params, Data extends state.State> {
  /**
   * The name of the resource being retrieve. This is used to make pretty messages for
   * the various query states.
   */
  name: string;
  /** Executed when the query is first created, or whenever the query parameters change. */
  retrieve: (args: RetrieveArgs<QParams>) => Promise<Data>;
  /**
   * Listeners to mount to the query. These listeners will be re-mounted when
   * the query parameters changed and/or the client disconnects/re-connects or clusters
   * are switched.
   *
   * These listeners will NOT be remounted when the identity of the onChange function
   * changes.
   */
  listeners?: ListenerConfig<QParams, Data>[];
  /**
   * A function that is called whenever the query result changes. This function is
   * responsible for updating the query state.
   */
  onChange: state.Setter<Result<Data>>;
  /**
   * The client to use to retrieve the resource. If the client is null, the query will
   * not be able to retrieve the resource.
   */
  client: Synnax | null;
}

interface UseObservableReturn<QueryParams extends Params> {
  retrieve: (params: QueryParams, options: { signal?: AbortSignal }) => void;
  retrieveAsync: (
    params: QueryParams,
    options: { signal?: AbortSignal },
  ) => Promise<void>;
}

/**
 * A low level hook that is used to create a query, and allows the caller to manage
 * the result state externally.
 *
 * @template QueryParams - The type of the parameters for the query.
 * @template Data - The type of the data being retrieved.
 */
