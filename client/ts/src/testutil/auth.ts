// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { url } from "@synnaxlabs/x";

import { auth } from "@/auth";
import { TEST_CLIENT_PARAMS } from "@/testutil/client";
import { Transport } from "@/transport";

/**
 * Creates a client that only logs in. It opens no streams, so a user with no role, or a
 * client whose key is deleted, can use it without stream errors.
 */
export const createTestAuthClient = (
  authority: auth.Authority,
  observer?: auth.Observer,
): auth.Client => {
  const transport = new Transport(
    new url.URL({
      host: TEST_CLIENT_PARAMS.host,
      port: Number(TEST_CLIENT_PARAMS.port),
    }),
  );
  const client = new auth.Client(transport.unary, authority, observer);
  transport.use(client.middleware());
  return client;
};
