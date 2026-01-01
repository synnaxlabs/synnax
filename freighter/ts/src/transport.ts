// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware } from "@/middleware";

/**
 * Transport is a based interface that represents a general transport for
 * exchanging messages between a client and a server.
 */
export interface Transport {
  /**
   * Use registers middleware that will be executed in order when the transport
   *
   * @param mw - The middleware to register.
   */
  use: (...mw: Middleware[]) => void;
}
