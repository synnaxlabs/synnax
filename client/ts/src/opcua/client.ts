// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { opcuaServer } from "@/opcua/server";

/** Client groups the clients of the OPC UA resources. */
export class Client {
  readonly servers: opcuaServer.Client;

  constructor(cfg: opcuaServer.ClientConfig) {
    this.servers = new opcuaServer.Client(cfg);
  }
}
