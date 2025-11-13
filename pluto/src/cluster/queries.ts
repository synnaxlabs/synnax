// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  checkConnection,
  type CheckConnectionParams,
  type connection,
  newConnectionChecker,
} from "@synnaxlabs/client";

import { Flux } from "@/flux";

export interface UseConnectionStateQuery extends CheckConnectionParams {}

export const { useRetrieve: useConnectionState } = Flux.createRetrieve<
  UseConnectionStateQuery,
  connection.State,
  {},
  true
>({
  name: "connectionState",
  allowDisconnected: true,
  retrieve: async ({ query }) => await checkConnection(query),
  mountListeners: ({ onChange, query }) => {
    const checker = newConnectionChecker(query);
    checker.onChange(onChange);
    return () => checker.stop();
  },
});
