// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Env, type Given, resolveParams } from "@/env";
import { ValidationError } from "@/errors";

const ENV = {
  SYNNAX_HOST: "core.env",
  SYNNAX_PORT: "9091",
  SYNNAX_API_KEY: "syk_env",
  SYNNAX_SECURE: "true",
};

const throwing = (error: Error): Env =>
  new Proxy(
    {},
    {
      get: () => {
        throw error;
      },
    },
  );

const notCapable = (): Error => {
  const error = new Error("Requires env access");
  error.name = "NotCapable";
  return error;
};

describe("resolveParams", () => {
  it.each<[string, Given, Env, Given]>([
    [
      "the variables fill every missing field",
      {},
      ENV,
      { host: "core.env", port: 9091, apiKey: "syk_env", secure: true },
    ],
    [
      "the params win over the variables",
      { host: "core.given", port: 80, apiKey: "syk_given", secure: false },
      ENV,
      { host: "core.given", port: 80, apiKey: "syk_given", secure: false },
    ],
    [
      "a variable fills a field the params leave out",
      { host: "rig", port: 9090 },
      ENV,
      { host: "rig", port: 9090, apiKey: "syk_env", secure: true },
    ],
    [
      "the defaults fill last",
      {},
      { SYNNAX_HOST: "core.env" },
      { host: "core.env", port: 9090, apiKey: undefined, secure: false },
    ],
    [
      "an empty host is unset",
      { host: "" },
      { SYNNAX_HOST: "core.env" },
      { host: "core.env", port: 9090, apiKey: undefined, secure: false },
    ],
    [
      "values are trimmed, and an empty value is unset",
      {},
      {
        SYNNAX_HOST: " core.env\n",
        SYNNAX_PORT: "",
        SYNNAX_API_KEY: "syk_env\n",
        SYNNAX_SECURE: " ",
      },
      { host: "core.env", port: 9090, apiKey: "syk_env", secure: false },
    ],
    [
      "credentials in the params keep SYNNAX_API_KEY out",
      { username: "u", password: "p" },
      ENV,
      {
        host: "core.env",
        port: 9091,
        username: "u",
        password: "p",
        apiKey: undefined,
        secure: true,
      },
    ],
    [
      "a token in the params keeps SYNNAX_API_KEY out",
      { host: "rig", token: "t" },
      ENV,
      { host: "rig", port: 9091, token: "t", apiKey: undefined, secure: true },
    ],
    [
      "a variable that Deno cannot read is unset",
      { host: "rig" },
      throwing(notCapable()),
      { host: "rig", port: 9090, apiKey: undefined, secure: false },
    ],
  ])("should resolve when %s", (_, params, env, expected) => {
    expect(resolveParams(params, env)).toEqual(expected);
  });

  it.each([
    ["1", true],
    ["yes", true],
    ["TRUE", true],
    ["0", false],
    ["no", false],
    ["false", false],
  ])("should read SYNNAX_SECURE=%j as %s", (value, secure) => {
    expect(resolveParams({ host: "rig" }, { SYNNAX_SECURE: value }).secure).toEqual(
      secure,
    );
  });

  it("should read no variable for a field the params give", () => {
    const params = { host: "rig", port: 80, apiKey: "k", secure: false };
    expect(resolveParams(params, throwing(new Error("read")))).toEqual(params);
  });

  it.each<[Given, Env, string]>([
    [{}, {}, "give a host, or set SYNNAX_HOST"],
    [{ apiKey: "k" }, { SYNNAX_HOST: " " }, "give a host, or set SYNNAX_HOST"],
    [{ host: "rig" }, { SYNNAX_PORT: "abc" }, "SYNNAX_PORT must be a port"],
    [{ host: "rig" }, { SYNNAX_PORT: "0" }, "SYNNAX_PORT must be a port"],
    [{ host: "rig" }, { SYNNAX_PORT: "65536" }, "SYNNAX_PORT must be a port"],
    [{ host: "rig" }, { SYNNAX_SECURE: "maybe" }, "SYNNAX_SECURE must be true or"],
  ])("should reject %j with %j", (params, env, message) => {
    expect(() => resolveParams(params, env)).toThrow(ValidationError);
    expect(() => resolveParams(params, env)).toThrow(message);
  });

  it("should reject a runtime with no environment variables and no host", () => {
    expect(() => resolveParams({}, undefined)).toThrow(ValidationError);
  });

  it("should let errors other than NotCapable propagate", () => {
    expect(() => resolveParams({}, throwing(new Error("read")))).toThrow("read");
  });
});
