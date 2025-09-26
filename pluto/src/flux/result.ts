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

export type InitialStatusDetailsContainer<StatusDetails = never> = [
  StatusDetails,
] extends [never]
  ? {}
  : { initialStatusDetails: StatusDetails };

export const parseInitialStatusDetails = <StatusDetails = never>(
  container: InitialStatusDetailsContainer<StatusDetails>,
): StatusDetails => {
  if ("initialStatusDetails" in container) return container.initialStatusDetails;
  // If status details are not in the container, this means that the caller did NOT
  // specify a type argument for the status details, so return undefined to make
  // type checking easier for the rest of the codebase.
  return undefined as StatusDetails;
};

export const resultStatusDetails = <Data extends state.State, StatusDetails = never>(
  result: Result<Data, StatusDetails>,
): StatusDetails => {
  if ("details" in result.status) return result.status.details as StatusDetails;
  return undefined as StatusDetails;
};

export type ResultStatus<StatusDetails = never> =
  | status.Status<StatusDetails, "success">
  | status.Status<StatusDetails, "loading">
  | status.Status<StatusDetails, "disabled">
  | status.Status<status.ExceptionDetails, "error">;

export interface ErrorResult {
  variant: "error";
  status: status.Status<status.ExceptionDetails, "error">;
  data: undefined;
}

export type SuccessResult<Data extends state.State, StatusDetails = never> = {
  variant: "success";
  status: status.Status<StatusDetails, "success">;
  data: Data;
};

export type LoadingResult<Data extends state.State, StatusDetails = never> = {
  variant: "loading";
  status: status.Status<StatusDetails, "loading">;
  data: Data | undefined;
};

export type DisabledResult<Data extends state.State, StatusDetails = never> = {
  variant: "disabled";
  status: status.Status<StatusDetails, "disabled">;
  data: Data | undefined;
};

export type Result<Data extends state.State, StatusDetails = never> =
  | ErrorResult
  | SuccessResult<Data, StatusDetails>
  | LoadingResult<Data, StatusDetails>
  | DisabledResult<Data, StatusDetails>;

interface ResultCreator {
  <Data extends state.State>(op: string, data?: Data | undefined): Result<Data, never>;
  <Data extends state.State, StatusDetails = never>(
    op: string,
    data: Data | undefined,
    statusDetails: StatusDetails,
  ): Result<Data, StatusDetails>;
}

export const loadingResult = (<Data extends state.State, StatusDetails = never>(
  op: string,
  data?: Data | undefined,
  statusDetails?: StatusDetails,
): LoadingResult<Data, StatusDetails> => ({
  variant: "loading",
  status: status.create<StatusDetails, "loading">({
    variant: "loading",
    message: `${caseconv.capitalize(op)}`,
    details: statusDetails as StatusDetails,
  }),
  data,
})) as ResultCreator;

export const successResult = (<Data extends state.State, StatusDetails = never>(
  op: string,
  data: Data,
  statusDetails: StatusDetails,
): SuccessResult<Data, StatusDetails> => ({
  variant: "success",
  status: status.create<StatusDetails, "success">({
    variant: "success",
    message: `Successfully ${op}`,
    details: statusDetails,
  }),
  data,
})) as ResultCreator;

export const errorResult = (op: string, error: unknown): ErrorResult => ({
  variant: "error",
  status: status.fromException(error, `Failed to ${op}`),
  data: undefined,
});

interface NullClientResultCreator {
  <Data extends state.State>(op: string): DisabledResult<Data, never>;
  <Data extends state.State, StatusDetails = never>(
    op: string,
    statusDetails: StatusDetails,
  ): DisabledResult<Data, StatusDetails>;
}

export const nullClientResult = (<Data extends state.State, StatusDetails = never>(
  op: string,
  statusDetails?: StatusDetails,
): DisabledResult<Data, StatusDetails> => ({
  variant: "disabled",
  status: status.create<StatusDetails, "disabled">({
    variant: "disabled",
    message: `Failed to ${op}`,
    description: `Cannot ${op} because no cluster is connected.`,
    details: statusDetails as StatusDetails,
  }),
  data: undefined,
})) as NullClientResultCreator;
