// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type framer } from "@synnaxlabs/client";
import { type Destructor, type observe } from "@synnaxlabs/x";

/**
 * Handler function for processing incoming frames from a streamer.
 */
export interface FrameHandler extends observe.Handler<framer.Frame> {}

/**
 * Configuration for subscribing to a channel stream.
 */
export interface Subscriber {
  /** The name of the channel to subscribe to */
  channel: channel.Name;
  /** Handler function called when frames are received */
  handler: FrameHandler;
  /** Optional callback invoked when the stream opens */
  onOpen?: () => void;
}

/**
 * Function type for adding a channel listener.
 * 
 * @param subscriber - The subscriber configuration
 * @returns A destructor function to remove the listener
 */
export interface ListenerAdder {
  (subscriber: Subscriber): Destructor;
}
