// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import Synnax, { type SynnaxParams } from "@/client";

export const TEST_CLIENT_PROPS: SynnaxParams = {
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

export const createTestClient = (props?: Partial<SynnaxParams>): Synnax =>
  new Synnax({ ...TEST_CLIENT_PROPS, ...props });
