// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { caseconv, type state } from "@synnaxlabs/x";
import type z from "zod";

/** Carries `initialStatusDetails` only when the query declares a details schema. */
export type InitialStatusDetailsContainer<
  StatusDetails extends z.ZodType = z.ZodNever,
> = [StatusDetails] extends [z.ZodNever]
  ? {}
  : { initialStatusDetails: z.output<StatusDetails> };

/** @returns the initial status details, or undefined when the query declares none. */
export const parseInitialStatusDetails = <StatusDetails extends z.ZodType = z.ZodNever>(
  container: InitialStatusDetailsContainer<StatusDetails>,
): z.output<StatusDetails> => {
  if ("initialStatusDetails" in container) return container.initialStatusDetails;
  // If status details are not in the container, this means that the caller did NOT
  // specify a type argument for the status details, so return undefined to make
  // type checking easier for the rest of the codebase.
  return undefined as z.output<StatusDetails>;
};

/** @returns the details a result's status carries, or undefined when it carries none. */
export const resultStatusDetails = <
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  result: Result<Data, StatusDetails>,
): z.output<StatusDetails> => {
  if ("details" in result.status)
    return result.status.details as z.output<StatusDetails>;
  return undefined as z.output<StatusDetails>;
};

/** The status a {@link Result} carries, narrowed by its variant. */
export type ResultStatus<StatusDetails extends z.ZodType = z.ZodNever> =
  | status.Status<StatusDetails, z.ZodLiteral<"success">>
  | status.Status<StatusDetails, z.ZodLiteral<"loading">>
  | status.Status<StatusDetails, z.ZodLiteral<"disabled">>
  | status.Status<typeof status.exceptionDetailsSchema, z.ZodLiteral<"error">>;

/** A query that failed. Its status holds the error and its stack. */
export interface ErrorResult {
  variant: "error";
  status: status.Status<typeof status.exceptionDetailsSchema, z.ZodLiteral<"error">>;
  data: undefined;
}

/** A query that returned. Its data is always present. */
export type SuccessResult<
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
> = {
  variant: "success";
  status: status.Status<StatusDetails, z.ZodLiteral<"success">>;
  data: Data;
};

/** A query still in flight. Its data holds the previous result, when there was one. */
export type LoadingResult<
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
> = {
  variant: "loading";
  status: status.Status<StatusDetails, z.ZodLiteral<"loading">>;
  data: Data | undefined;
};

/** A query that never ran, because no Core is connected or no query was given. */
export type DisabledResult<
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
> = {
  variant: "disabled";
  status: status.Status<StatusDetails, z.ZodLiteral<"disabled">>;
  data: Data | undefined;
};

/**
 * The state of a Flux query. Switch on `variant` to narrow it: only a success result
 * guarantees its data.
 */
export type Result<
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
> =
  | ErrorResult
  | SuccessResult<Data, StatusDetails>
  | LoadingResult<Data, StatusDetails>
  | DisabledResult<Data, StatusDetails>;

interface LoadingResultCreator {
  <Data extends state.State>(
    op: string,
    data?: Data | undefined,
  ): LoadingResult<Data, never>;
  <Data extends state.State, StatusDetails extends z.ZodType = z.ZodNever>(
    op: string,
    data: Data | undefined,
    statusDetails: z.output<StatusDetails>,
  ): LoadingResult<Data, StatusDetails>;
}

interface SuccessResultCreator {
  <Data extends state.State>(
    op: string,
    data?: Data | undefined,
  ): SuccessResult<Data, never>;
  <Data extends state.State, StatusDetails extends z.ZodType = z.ZodNever>(
    op: string,
    data: Data | undefined,
    statusDetails: z.output<StatusDetails>,
  ): SuccessResult<Data, StatusDetails>;
}

/** Builds a {@link LoadingResult} whose message names the operation in flight. */
export const loadingResult = (<
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  op: string,
  data?: Data | undefined,
  statusDetails?: StatusDetails,
): LoadingResult<Data, StatusDetails> => ({
  variant: "loading",
  status: status.create<StatusDetails, "loading">({
    variant: "loading",
    message: `${caseconv.capitalize(op)}`,
    details: statusDetails as z.output<StatusDetails>,
  }),
  data,
})) as LoadingResultCreator;

/** Builds a {@link SuccessResult} whose message names the operation that finished. */
export const successResult = (<
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  op: string,
  data: Data,
  statusDetails: z.output<StatusDetails>,
): SuccessResult<Data, StatusDetails> => ({
  variant: "success",
  status: status.create<StatusDetails, "success">({
    variant: "success",
    message: caseconv.capitalize(op),
    details: statusDetails,
  }),
  data,
})) as SuccessResultCreator;

/** Builds an {@link ErrorResult} from a thrown value, keeping its cause chain. */
export const errorResult = (op: string, error: unknown): ErrorResult => ({
  variant: "error",
  status: status.fromException(error, `Failed to ${op}`),
  data: undefined,
});

interface NullClientResultCreator {
  <Data extends state.State>(op: string): DisabledResult<Data, never>;
  <Data extends state.State, StatusDetails extends z.ZodType = z.ZodNever>(
    op: string,
    statusDetails: z.output<StatusDetails>,
  ): DisabledResult<Data, StatusDetails>;
}

/** Builds a {@link DisabledResult} for a query attempted with no Core connected. */
export const nullClientResult = (<
  Data extends state.State,
  StatusDetails extends z.ZodType = z.ZodNever,
>(
  op: string,
  statusDetails?: z.output<StatusDetails>,
): DisabledResult<Data, StatusDetails> => ({
  variant: "disabled",
  status: status.create<StatusDetails, "disabled">({
    variant: "disabled",
    message: `Failed to ${op}`,
    description: "No Core is connected.",
    details: statusDetails as z.output<StatusDetails>,
  }),
  data: undefined,
})) as NullClientResultCreator;

/** Builds a {@link DisabledResult} for a query the caller left unset. */
export const noQueryResult = <Data extends state.State>(
  op: string,
): DisabledResult<Data, never> => ({
  variant: "disabled",
  status: status.create<never, "disabled">({
    variant: "disabled",
    message: `Did not ${op}`,
    description: `Cannot ${op} without a query.`,
  }),
  data: undefined,
});
