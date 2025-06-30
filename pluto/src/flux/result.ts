// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x";

import { type state } from "@/state";

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
