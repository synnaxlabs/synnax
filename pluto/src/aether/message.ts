// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type errors } from "@synnaxlabs/x";

import { type state } from "@/state";

/** A message from the main thread to update or create an aether component. */
export interface MainUpdateMessage {
  variant: "update";
  /** The path of the component to update. */
  path: string[];
  /** The type of the component to update. */
  type: string;
  /** The state of the component to update. */
  state: state.State;
}

/** A message from the main thread to delete an aether component. */
export interface MainDeleteMessage {
  variant: "delete";
  /** The type of the component to delete. */
  type: string;
  /** The path of the component to delete. */
  path: string[];
}

/** A message from the aether thread to update an aether component. */
export interface AetherUpdateMessage {
  variant: "update";
  /** The key of the component to update. */
  key: string;
  /** The state of the component to update. */
  state: state.State;
}

/** A message from the aether thread to send an error to the main thread. */
export interface AetherErrorMessage {
  variant: "error";
  error: errors.NativePayload;
}

/** A message from the aether thread to the main thread. */
export type AetherMessage = AetherUpdateMessage | AetherErrorMessage;

/** A message from the main thread to the aether thread. */
export type MainMessage = MainUpdateMessage | MainDeleteMessage;
