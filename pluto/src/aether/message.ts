// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type errors } from "@synnaxlabs/x";

/** @description A message from the main thread to update or create an aether component. */
export interface MainUpdateMessage {
  variant: "update";
  /** @description The path of the component to update. */
  path: string[];
  /** @description The type of the component to update. */
  type: string;
  /** @description The state of the component to update. */
  state: any;
}

/** @description A message from the main thread to delete an aether component. */
export interface MainDeleteMessage {
  variant: "delete";
  /** @description The type of the component to delete. */
  type: string;
  /** @description The path of the component to delete. */
  path: string[];
}

/** @description A message from the aether thread to update an aether component. */
export interface AetherUpdateMessage {
  variant: "update";
  /** @description The key of the component to update. */
  key: string;
  /** @description The state of the component to update. */
  state: any;
}

/** @description A message from the aether thread to send an error to the main thread. */
export interface AetherErrorMessage {
  variant: "error";
  error: errors.NativePayload;
}

/** @description A message from the aether thread to the main thread. */
export type AetherMessage = AetherUpdateMessage | AetherErrorMessage;

/** @description A message from the main thread to the aether thread. */
export type MainMessage = MainUpdateMessage | MainDeleteMessage;
