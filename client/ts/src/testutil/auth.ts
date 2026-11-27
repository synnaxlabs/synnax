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
import { testClientParams } from "@/testutil/client";
import { Transport } from "@/transport";

/** Creates an auth client that logs in and opens no streams. */
export const createTestAuthClient = (
  authority: auth.Authority,
  observer?: auth.Observer,
): auth.Client => {
  const { host, port } = testClientParams();
  const transport = new Transport({
    url: new url.URL({ host, port: Number(port) }),
    secure: false,
  });
  const client = new auth.Client(transport.unary, authority, observer);
  transport.use(client.middleware());
  return client;
};
