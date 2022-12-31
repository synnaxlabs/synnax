// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { URL } from "@synnaxlabs/freighter";
import { describe, expect, test } from "vitest";

import { HOST, PORT } from "../setupspecs";

import AuthenticationClient from "./auth";
import { AuthError } from "./errors";
import Transport from "./transport";

describe("auth", () => {
  test("valid credentials", async () => {
    const transport = new Transport(new URL({ host: HOST, port: PORT }));
    const client = new AuthenticationClient(transport.httpFactory, {
      username: "synnax",
      password: "seldon",
    });
    await client.authenticating;
    expect(client.authenticated).toBeTruthy();
  });

  test("invalid credentials", async () => {
    const transport = new Transport(new URL({ host: HOST, port: PORT }));
    const client = new AuthenticationClient(transport.httpFactory, {
      username: "synnax",
      password: "wrong",
    });
    try {
      await client.authenticating;
      expect(client.authenticated).toBeFalsy();
    } catch (e) {
      expect(client.authenticated).toBeFalsy();
      expect(e).toBeInstanceOf(AuthError);
      if (e instanceof AuthError) {
        expect(e.message).toEqual("[synnax] - invalid credentials");
      }
    }
  });
});
