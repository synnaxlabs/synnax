// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { afterAll } from "vitest";

import Synnax, { type SynnaxParams } from "@/client";

export const TEST_CLIENT_PARAMS: SynnaxParams = {
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  retry: {
    maxRetries: 4,
    baseInterval: TimeSpan.seconds(1),
    scale: 1.5,
  },
};

// A hook registered mid-test is silently dropped by vitest, so per-client afterAll
// calls leak clients created inside test bodies. This module-level hook registers
// during spec collection (import time) and covers every call site.
const openClients: Synnax[] = [];
afterAll(() => openClients.forEach((client) => client.close()));

/**
 * Creates a client connected to the local test cluster. The client is closed
 * automatically once every test in the current spec file has finished, so callers do
 * not need to close it themselves.
 */
export const createTestClient = (params?: Partial<SynnaxParams>): Synnax => {
  const client = new Synnax({ ...TEST_CLIENT_PARAMS, ...params });
  openClients.push(client);
  return client;
};
