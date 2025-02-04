// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StateTracker } from "@/control/state";
import { type framer } from "@/framer";

const CONTROL_STATE_KEY = "sy_node_1_control";

export class Client {
  private readonly framer: framer.Client;

  constructor(framer: framer.Client) {
    this.framer = framer;
  }

  async openStateTracker(): Promise<StateTracker> {
    const stream = await this.framer.openStreamer(CONTROL_STATE_KEY);
    return new StateTracker(stream);
  }
}
