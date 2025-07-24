// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { caseconv, status } from "@synnaxlabs/x";

import { type state } from "@/state";

/**
 * The result of a query operation that can be in one of three states:
 * - `loading` - Data is currently being fetched from the server
 * - `error` - An error occurred while fetching data, or while handling values provided
 *   to a listener
 * - `success` - Data was successfully fetched and is available in the `data` field
 *
 * @template Data The type of data being retrieved, must extend state.State
 *
 * @example
 * ```typescript
 * const result: Result<User[]> = {
 *   variant: "success",
 *   data: [{ id: 1, name: "John" }],
 *   error: null,
 *   message: "Retrieved users"
 * };
 * ```
 */
export type Result<Data extends state.State> =
  | (status.Status<status.ExceptionDetails, "error"> & {
      /** The data payload, null when in error state */
      data: null;
    })
  | (status.Status<undefined, "loading"> & {
      /** The data payload, may be null or contain previous data while loading */
      data: null | Data;
    })
  | (status.Status<undefined, "success"> & {
      /** The successfully retrieved data */
      data: Data;
    });

/**
 * Factory function to create a loading result state.
 *
 * @template Data The type of data being retrieved
 * @param name The name of the resource being retrieved (used in status messages)
 * @param op The operation being performed (e.g., "retrieving", "updating")
 * @param data Optional existing data to preserve during loading
 * @returns A Result object in loading state
 *
 * @example
 * ```typescript
 * const loadingResult = pendingResult<User[]>("users", "retrieving");
 * // Returns: { variant: "loading", data: null, error: null, message: "Retrieving users" }
 * ```
 */
export const pendingResult = <Data extends state.State>(
  name: string,
  op: string,
  data: Data | null = null,
): Result<Data> => ({
  ...status.create<undefined, "loading">({
    variant: "loading",
    message: `${caseconv.capitalize(op)} ${name}`,
  }),
  data,
});

/**
 * Factory function to create a success result state.
 *
 * @template Data The type of data being retrieved
 * @param name The name of the resource being retrieved (used in status messages)
 * @param op The operation that was performed (e.g., "retrieved", "updated")
 * @param data The successfully retrieved data
 * @returns A Result object in success state
 *
 * @example
 * ```typescript
 * const successResult = successResult<User[]>("users", "retrieved", userList);
 * // Returns: { variant: "success", data: userList, error: null, message: "Retrieved users" }
 * ```
 */
export const successResult = <Data extends state.State>(
  name: string,
  op: string,
  data: Data,
): Result<Data> => ({
  ...status.create<undefined, "success">({
    variant: "success",
    message: `${caseconv.capitalize(op)} ${name}`,
  }),
  data,
});

/**
 * Factory function to create an error result state.
 *
 * @template Data The type of data being retrieved
 * @param name The name of the resource being retrieved (used in status messages)
 * @param op The operation that failed (e.g., "retrieve", "update")
 * @param error The error that occurred
 * @returns A Result object in error state
 *
 * @example
 * ```typescript
 * const errorResult = errorResult<User[]>("users", "retrieve", new Error("Network error"));
 * // Returns: { variant: "error", data: null, error: Error, message: "Failed to retrieve users" }
 * ```
 */
export const errorResult = <Data extends state.State>(
  name: string,
  op: string,
  error: unknown,
): Result<Data> => ({
  ...status.fromException(error, `Failed to ${op} ${name}`),
  data: null,
});

/**
 * Factory function to create an error result for operations that require a connected client.
 *
 * @template Data The type of data being retrieved
 * @param name The name of the resource being retrieved
 * @param opName The operation that cannot be performed
 * @returns A Result object in error state with a DisconnectedError
 *
 * @example
 * ```typescript
 * const result = nullClientResult<User[]>("users", "retrieve");
 * // Returns error result with message: "Cannot retrieve users because no cluster is connected."
 * ```
 */
export const nullClientResult = <Data extends state.State>(
  name: string,
  opName: string,
): Result<Data> =>
  errorResult(
    name,
    opName,
    new DisconnectedError(`Cannot ${opName} ${name} because no cluster is connected.`),
  );
