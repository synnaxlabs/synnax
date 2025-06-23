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

export interface FrameHandler extends observe.Handler<framer.Frame> {}

export interface Subscriber {
  channels: channel.Name | channel.Names;
  handler: FrameHandler;
}

export interface ListenerAdder {
  (subscriber: Subscriber): Destructor;
}
