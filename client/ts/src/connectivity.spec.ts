// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { URL } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { AuthenticationClient } from "@/auth";
import { ConnectivityClient } from "@/connectivity";
import { HOST, PORT } from "@/setupspecs";
import { Transport } from "@/transport";

describe("connectivity", () => {
  it("should connect to the server", async () => {
    const transport = new Transport(new URL({ host: HOST, port: PORT }));
    const client = new AuthenticationClient(transport.unary, {
      username: "synnax",
      password: "seldon",
    });
    await client.authenticating;
    expect(client.authenticated).toBeTruthy();

    transport.use(client.middleware());

    const connectivity = new ConnectivityClient(transport.unary);

    await connectivity.check();
    expect(connectivity.status()).toEqual("connected");
  });
});
