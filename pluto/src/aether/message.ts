// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface MainUpdate {
  variant: "update";
  path: string[];
  type: string;
  state: any;
}

export interface MainDelete {
  variant: "delete";
  type: string;
  path: string[];
}

export interface WorkerUpdate {
  variant: "update";
  key: string;
  state: any;
}

export interface ErrorObject {
  name: string;
  message: string;
  stack?: string;
}

export interface WorkerError {
  variant: "error";
  error: ErrorObject;
}

export type WorkerMessage = WorkerUpdate | WorkerError;
export type MainMessage = MainUpdate | MainDelete;
