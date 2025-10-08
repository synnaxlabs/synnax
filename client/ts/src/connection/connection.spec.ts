// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { URL } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { auth } from "@/auth";
import { connection } from "@/connection";
import { TEST_CLIENT_PROPS } from "@/testutil/client";
import { Transport } from "@/transport";

describe("connectivity", () => {
  it("should connect to the server", async () => {
    const transport = new Transport(
      new URL({
        host: TEST_CLIENT_PROPS.host,
        port: Number(TEST_CLIENT_PROPS.port),
      }),
    );
    const client = new auth.Client(transport.unary, TEST_CLIENT_PROPS);
    transport.use(client.middleware());
    const connectivity = new connection.Checker(
      transport.unary,
      undefined,
      __VERSION__,
    );
    const state = await connectivity.check();
    expect(state.status).toEqual("connected");
    expect(z.uuid().safeParse(state.clusterKey).success).toBe(true);
  });
  describe("version compatibility", () => {
    it("should pull the server and client versions", async () => {
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PROPS.host,
          port: Number(TEST_CLIENT_PROPS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PROPS);
      transport.use(client.middleware());
      const connectivity = new connection.Checker(
        transport.unary,
        undefined,
        __VERSION__,
      );
      const state = await connectivity.check();
      expect(state.clientServerCompatible).toBe(true);
      expect(state.clientVersion).toBe(__VERSION__);
    });
    it("should adjust state if the server is too old", async () => {
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PROPS.host,
          port: Number(TEST_CLIENT_PROPS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PROPS);
      transport.use(client.middleware());
      const connectivity = new connection.Checker(
        transport.unary,
        undefined,
        "50000.0.0",
      );
      const state = await connectivity.check();
      expect(state.clientServerCompatible).toBe(false);
      expect(state.clientVersion).toBe("50000.0.0");
    });
    it("should adjust state if the server is too new", async () => {
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PROPS.host,
          port: Number(TEST_CLIENT_PROPS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PROPS);
      transport.use(client.middleware());
      const connectivity = new connection.Checker(transport.unary, undefined, "0.0.0");
      const state = await connectivity.check();
      expect(state.clientServerCompatible).toBe(false);
      expect(state.clientVersion).toBe("0.0.0");
    });
  });
});
