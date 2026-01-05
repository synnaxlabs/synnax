// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StateTracker } from "@/control/state";
import { framer } from "@/framer";

export const CONTROL_STATE_CHANNEL_NAME = "sy_node_1_control";

export class Client {
  private readonly framer: framer.Client;

  constructor(framer: framer.Client) {
    this.framer = framer;
  }

  async openStateTracker(): Promise<StateTracker> {
    const stream = await framer.HardenedStreamer.open(
      async (p) => await this.framer.openStreamer(p),
      CONTROL_STATE_CHANNEL_NAME,
    );
    return new StateTracker(stream);
  }
}
