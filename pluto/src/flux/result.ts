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
 * The result of a query. The query can be in one of three states:
 * - `loading` - Data is currently being fetched.
 * - `error` - An error occurred while fetching data, or while handling values provided
 * to a listener.
 * - `success` - Data was successfully fetched and is available in the `data` field.
 */
export type Result<Data extends state.State> =
  | (status.Status<status.ExceptionDetails, "error"> & {
      data: null;
      error: unknown;
    })
  | (status.Status<undefined, "loading"> & {
      data: null | Data;
      error: null;
    })
  | (status.Status<undefined, "success"> & {
      data: Data;
      error: null;
    });

/** A factory function to create a loading result. */
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
  error: null,
});

/** A factory function to create a success result. */
export const successResult = <Data extends state.State>(
  name: string,
  op: string,
  value: Data,
): Result<Data> => ({
  ...status.create<undefined, "success">({
    variant: "success",
    message: `${caseconv.capitalize(op)} ${name}`,
  }),
  data: value,
  error: null,
});

/** A factory function to create an error result. */
export const errorResult = <Data extends state.State>(
  name: string,
  op: string,
  error: unknown,
): Result<Data> => ({
  ...status.fromException(error, `Failed to ${op} ${name}`),
  data: null,
  error,
});

export const nullClientResult = <Data extends state.State>(
  name: string,
  opName: string,
): Result<Data> =>
  errorResult(
    name,
    opName,
    new DisconnectedError(`Cannot ${opName} ${name} because no cluster is connected.`),
  );
