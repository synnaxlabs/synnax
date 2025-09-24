// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, status } from "@synnaxlabs/x";

import { type state } from "@/state";

export type StatusDataContainer<StatusData = never> = [StatusData] extends [never]
  ? {}
  : { statusData: StatusData };

export const parseStatusData = <StatusData = never>(
  container: StatusDataContainer<StatusData>,
): StatusData => {
  if ("statusData" in container) return container.statusData;
  return undefined as StatusData;
};

export type Status<StatusData = never> =
  | status.Status<StatusData, "success">
  | status.Status<StatusData, "loading">
  | status.Status<StatusData, "disabled">
  | status.Status<status.ExceptionDetails, "error">;

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
export type ErrorResult = {
  variant: "error";
  status: status.Status<status.ExceptionDetails, "error">;
  data: undefined;
};

export type SuccessResult<Data extends state.State, StatusData = never> = {
  variant: "success";
  status: status.Status<StatusData, "success">;
  data: Data;
};

export type LoadingResult<Data extends state.State, StatusData = never> = {
  variant: "loading";
  status: status.Status<StatusData, "loading">;
  data: Data | undefined;
};

export type DisabledResult<Data extends state.State, StatusData = never> = {
  variant: "disabled";
  status: status.Status<StatusData, "disabled">;
  data: Data | undefined;
};

export type Result<Data extends state.State, StatusData = never> =
  | ErrorResult
  | SuccessResult<Data, StatusData>
  | LoadingResult<Data, StatusData>
  | DisabledResult<Data, StatusData>;

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

interface LoadingResultCreator {
  <Data extends state.State>(
    op: string,
    data?: Data | undefined,
  ): LoadingResult<Data, never>;
  <Data extends state.State, StatusData = never>(
    op: string,
    data: Data | undefined,
    statusData: StatusData,
  ): LoadingResult<Data, StatusData>;
}

export const loadingResult = (<Data extends state.State, StatusData = never>(
  op: string,
  data: Data | undefined = undefined,
  statusData?: StatusData,
): LoadingResult<Data, StatusData> => ({
  variant: "loading",
  status: status.create<StatusData, "loading">({
    variant: "loading",
    message: `${caseconv.capitalize(op)}`,
    details: statusData as StatusData,
  }),
  data,
})) as LoadingResultCreator;

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

export interface SuccessResultCreator {
  <Data extends state.State>(op: string, data: Data): SuccessResult<Data, never>;
  <Data extends state.State, StatusData = never>(
    op: string,
    data: Data,
    statusData: StatusData,
  ): SuccessResult<Data, StatusData>;
}

export const successResult = (<Data extends state.State, StatusData = never>(
  op: string,
  data: Data,
  statusData: StatusData,
): SuccessResult<Data, StatusData> => ({
  variant: "success",
  status: status.create<StatusData, "success">({
    variant: "success",
    message: `Successfully ${op}`,
    details: statusData,
  }),
  data,
})) as SuccessResultCreator;

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
export const errorResult = (op: string, error: unknown): ErrorResult => ({
  variant: "error",
  status: status.fromException(error, `Failed to ${op}`),
  data: undefined,
});

interface NullClientResultCreator {
  <Data extends state.State>(op: string): DisabledResult<Data, never>;
  <Data extends state.State, StatusData = never>(
    op: string,
    statusData: StatusData,
  ): DisabledResult<Data, StatusData>;
}

export const nullClientResult = (<Data extends state.State, StatusData = never>(
  op: string,
  statusData?: StatusData,
): DisabledResult<Data, StatusData> => ({
  variant: "disabled",
  status: status.create<StatusData, "disabled">({
    variant: "disabled",
    message: `Failed to ${op}`,
    description: `Cannot ${op} because no cluster is connected.`,
    details: statusData as StatusData,
  }),
  data: undefined,
})) as NullClientResultCreator;
