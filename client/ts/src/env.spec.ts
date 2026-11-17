// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { loadParams } from "@/env";
import { ValidationError } from "@/errors";

const ENV = { SYNNAX_HOST: "core.lab", SYNNAX_API_KEY: "syk_env" };

describe("loadParams", () => {
  it("should use port 9090 and no TLS when only the host and key are set", () => {
    expect(loadParams(ENV)).toEqual({
      host: "core.lab",
      port: 9090,
      apiKey: "syk_env",
      secure: false,
    });
  });

  it("should read the port and TLS", () => {
    const params = loadParams({ ...ENV, SYNNAX_PORT: "9091", SYNNAX_SECURE: "TRUE" });
    expect(params).toMatchObject({ port: 9091, secure: true });
  });

  it("should trim values, such as the newline that ends a secret file", () => {
    expect(
      loadParams({
        SYNNAX_HOST: " core.lab\n",
        SYNNAX_API_KEY: "syk_env\n",
        SYNNAX_SECURE: "true ",
      }),
    ).toEqual({ host: "core.lab", port: 9090, apiKey: "syk_env", secure: true });
  });

  it("should use port 9090 when SYNNAX_PORT is empty", () => {
    expect(loadParams({ ...ENV, SYNNAX_PORT: "" }).port).toEqual(9090);
  });

  it.each([
    ["1", true],
    ["yes", true],
    ["0", false],
    ["false", false],
    ["", false],
  ])("should read SYNNAX_SECURE=%j as %s", (value, secure) => {
    expect(loadParams({ ...ENV, SYNNAX_SECURE: value }).secure).toEqual(secure);
  });

  it.each([
    [{}, "give a host, or set SYNNAX_HOST and SYNNAX_API_KEY"],
    [{ SYNNAX_HOST: "  ", SYNNAX_PORT: "1" }, "give a host, or set SYNNAX_HOST"],
    [{ SYNNAX_HOST: "h" }, "set SYNNAX_API_KEY with SYNNAX_HOST"],
    [{ SYNNAX_API_KEY: "k" }, "set SYNNAX_HOST with SYNNAX_API_KEY"],
    [{ ...ENV, SYNNAX_PORT: "abc" }, "SYNNAX_PORT must be a port"],
    [{ ...ENV, SYNNAX_PORT: "0" }, "SYNNAX_PORT must be a port"],
    [{ ...ENV, SYNNAX_PORT: "65536" }, "SYNNAX_PORT must be a port"],
    [{ ...ENV, SYNNAX_SECURE: "maybe" }, "SYNNAX_SECURE must be true or false"],
  ])("should reject %j", (env, message) => {
    expect(() => loadParams(env)).toThrow(ValidationError);
    expect(() => loadParams(env)).toThrow(message);
  });

  it("should reject a runtime with no environment variables", () => {
    expect(() => loadParams(undefined)).toThrow(ValidationError);
  });
});
