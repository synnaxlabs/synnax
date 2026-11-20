// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { niDevice } from "@/ni/device";

/** Client groups the clients of the NI resources. */
export class Client {
  readonly devices: niDevice.Client;

  constructor(cfg: niDevice.ClientConfig) {
    this.devices = new niDevice.Client(cfg);
  }
}
