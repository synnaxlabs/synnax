// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const START_COMMAND = "start";
export type StartCommand = typeof START_COMMAND;
export const STOP_COMMAND = "stop";
export type StopCommand = typeof STOP_COMMAND;
export type StartOrStopCommand = StartCommand | StopCommand;

export const RUNNING_STATUS = "running";
export type RunningStatus = typeof RUNNING_STATUS;
export const PAUSED_STATUS = "paused";
export type PausedStatus = typeof PAUSED_STATUS;
export const LOADING_STATUS = "loading";
export type LoadingStatus = typeof LOADING_STATUS;
export type Status = RunningStatus | PausedStatus | LoadingStatus;
