// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface UpdateMessage {
  variant: "update";
  path: string[];
  type: string;
  state: any;
}

export interface DeleteMessage {
  variant: "delete";
  path: string[];
}

export interface BackwardUpate {
  variant: "backward";
  key: string;
  state: any;
}

export type WorkerMessage = UpdateMessage | DeleteMessage | BackwardUpate;
