// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { chmodSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { Unreachable } from "@synnaxlabs/freighter";
import { describe, expect, it, vi } from "vitest";

import {
  type Env,
  type Given,
  nodeSystem,
  resolveParams,
  type Run,
  type System,
} from "@/env";
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

  it.each<[Given, Env, boolean]>([
    [{ caCert: "pem" }, {}, true],
    [{ caCert: "pem" }, { SYNNAX_SECURE: "false" }, true],
    [{ caCert: "pem", secure: false }, { SYNNAX_SECURE: "true" }, false],
    [{ secure: false }, { SYNNAX_SECURE: "true" }, false],
  ])("should take secure and the CA from %j over %j", (params, env, secure) => {
    expect(resolveParams({ host: "rig", ...params }, env).secure).toEqual(secure);
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

const DESKTOP_FILE = {
  host: "localhost",
  port: 9095,
  secure: true,
  api_key_helper: "/app/synnax-desktop-key",
};

const desktop = (
  file: object | null = DESKTOP_FILE,
  run: Run | null = { status: 0, stdout: "syk_desktop\n", stderr: "" },
): System => ({
  readFile: () => (file == null ? null : JSON.stringify(file)),
  run: vi.fn(() => run),
});

describe("resolveParams with Synnax Desktop", () => {
  it("should fill every missing field from Synnax Desktop", () => {
    const system = desktop();
    expect(resolveParams({}, {}, system)).toEqual({
      host: "localhost",
      port: 9095,
      apiKey: "syk_desktop",
      secure: true,
    });
    expect(system.run).toHaveBeenCalledWith("/app/synnax-desktop-key");
  });

  it("should let the params and variables win over Synnax Desktop", () => {
    const system = desktop();
    const env = { SYNNAX_SECURE: "false", SYNNAX_API_KEY: "syk_env" };
    expect(resolveParams({ port: 80 }, env, system)).toEqual({
      host: "localhost",
      port: 80,
      apiKey: "syk_env",
      secure: false,
    });
    expect(system.run).not.toHaveBeenCalled();
  });

  it("should take secure from caCert over Synnax Desktop", () => {
    const system = desktop({ ...DESKTOP_FILE, secure: false });
    expect(resolveParams({ caCert: "pem" }, {}, system)).toMatchObject({
      host: "localhost",
      port: 9095,
      caCert: "pem",
      secure: true,
    });
  });

  it("should not run the key helper when the params give credentials", () => {
    const system = desktop();
    const params = { username: "u", password: "p" };
    expect(resolveParams(params, {}, system).apiKey).toBeUndefined();
    expect(system.run).not.toHaveBeenCalled();
  });

  it("should not read Synnax Desktop when the params give a host", () => {
    const system: System = {
      readFile: () => {
        throw new Error("read");
      },
      run: () => {
        throw new Error("run");
      },
    };
    expect(resolveParams({ host: "rig" }, {}, system).host).toEqual("rig");
  });

  const thrown = (system: System): unknown => {
    try {
      resolveParams({}, {}, system);
    } catch (e) {
      return e;
    }
    throw new Error("resolveParams did not throw");
  };

  const reading = (error: Error): System => ({
    readFile: () => {
      throw error;
    },
    run: () => null,
  });

  it("should not name Synnax Desktop when there is no file", () => {
    const error = thrown(desktop(null));
    expect(error).toBeInstanceOf(ValidationError);
    expect(error).toHaveProperty("message", "give a host, or set SYNNAX_HOST");
  });

  it("should not name Synnax Desktop when Deno cannot read the file", () => {
    const cause = notCapable();
    const error = thrown(reading(cause));
    expect(error).toBeInstanceOf(ValidationError);
    expect(error).toHaveProperty("message", "give a host, or set SYNNAX_HOST");
    expect(error).toHaveProperty("cause", cause);
  });

  it("should name the file when it cannot be read", () => {
    const cause = new Error("EACCES: permission denied");
    const error = thrown(reading(cause));
    expect(error).toBeInstanceOf(ValidationError);
    expect(error).toHaveProperty(
      "message",
      "Cannot read ~/.synnax/desktop.json: EACCES: permission denied",
    );
    expect(error).toHaveProperty("cause", cause);
  });

  it("should name Synnax Desktop when Deno cannot run the key helper", () => {
    const cause = notCapable();
    const system: System = {
      readFile: () => JSON.stringify(DESKTOP_FILE),
      run: () => {
        throw cause;
      },
    };
    const error = thrown(system);
    expect(error).toBeInstanceOf(ValidationError);
    expect(error).toHaveProperty(
      "message",
      "Cannot run the key helper of Synnax Desktop: Requires env access",
    );
    expect(error).toHaveProperty("cause", cause);
  });

  it("should throw Unreachable when Synnax Desktop is not running", () => {
    const error = thrown(desktop(DESKTOP_FILE, null));
    if (!(error instanceof Unreachable)) throw new Error("expected Unreachable");
    expect(error.message).toEqual(
      "Synnax Desktop is not running. Open it and try again. To connect to another " +
        "Core, give a host and a port.",
    );
    expect(error.url.host).toEqual("localhost");
    expect(error.url.port).toEqual(9095);
  });

  it.each<[string, System, string]>([
    [
      "an older Synnax Desktop saved the key in the file",
      desktop({ ...DESKTOP_FILE, api_key: "syk_old" }),
      "Synnax Desktop saved its API key in an older format",
    ],
    [
      "the file has no key helper",
      desktop({ host: "localhost", port: 9095, secure: false }),
      "Cannot read ~/.synnax/desktop.json. Update Synnax Desktop and open it.",
    ],
    [
      "the file is not valid JSON",
      { readFile: () => "{", run: () => null },
      "Cannot read ~/.synnax/desktop.json",
    ],
    [
      "the key helper fails",
      desktop(DESKTOP_FILE, {
        status: 1,
        stdout: "",
        stderr: "failed to read the Synnax Desktop API key: no keychain\n",
      }),
      "failed to read the Synnax Desktop API key: no keychain",
    ],
  ])("should reject when %s", (_, system, message) => {
    expect(() => resolveParams({}, {}, system)).toThrow(ValidationError);
    expect(() => resolveParams({}, {}, system)).toThrow(message);
  });

  describe.skipIf(process.platform === "win32")("with a real key helper", () => {
    const withDir = (f: (dir: string, system: System) => void): void => {
      const dir = mkdtempSync(join(tmpdir(), "synnax-desktop-"));
      try {
        const system = nodeSystem();
        if (system == null) throw new Error("no system");
        f(dir, system);
      } finally {
        rmSync(dir, { recursive: true });
      }
    };

    const writeHelper = (dir: string, mode: number): string => {
      const helper = join(dir, "synnax-desktop-key");
      writeFileSync(helper, "#!/bin/sh\necho syk_real\n");
      chmodSync(helper, mode);
      return helper;
    };

    it("should read the key from the helper", () =>
      withDir((dir, real) => {
        const helper = writeHelper(dir, 0o755);
        const system: System = {
          readFile: () => JSON.stringify({ ...DESKTOP_FILE, api_key_helper: helper }),
          run: real.run,
        };
        expect(resolveParams({}, {}, system).apiKey).toEqual("syk_real");
      }));

    it("should return null for a helper that is not there", () =>
      withDir((dir, real) => {
        expect(real.run(join(dir, "missing"))).toBeNull();
      }));

    it("should name a helper that exists but cannot start", () =>
      withDir((dir, real) => {
        const helper = writeHelper(dir, 0o644);
        expect(() => real.run(helper)).toThrow(ValidationError);
        expect(() => real.run(helper)).toThrow(
          `Cannot run the key helper of Synnax Desktop at ${helper}`,
        );
      }));
  });
});

const PEM = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n";

// Reads only the files in the map, and runs no program.
const files = (map: Record<string, string>): System => ({
  readFile: (path) => map[path] ?? null,
  run: () => null,
});

describe("resolveParams with SYNNAX_CA_CERT", () => {
  const system = files({
    "~/ca.pem": PEM,
    "/empty.pem": " \n",
    "/ca.der": "0\u0082\u0001",
  });

  it("should turn on TLS with the PEM text of the file", () => {
    const env = { SYNNAX_CA_CERT: " ~/ca.pem\n" };
    expect(resolveParams({ host: "rig" }, env, system)).toMatchObject({
      secure: true,
      caCert: PEM,
    });
  });

  it("should take SYNNAX_SECURE=true with the file", () => {
    const env = { SYNNAX_SECURE: "true", SYNNAX_CA_CERT: "~/ca.pem" };
    expect(resolveParams({ host: "rig" }, env, system)).toMatchObject({
      secure: true,
      caCert: PEM,
    });
  });

  it("should take caCert in the params over the variable", () => {
    const env = { SYNNAX_CA_CERT: "/missing.pem" };
    expect(resolveParams({ host: "rig", caCert: "pem" }, env, system)).toMatchObject({
      secure: true,
      caCert: "pem",
    });
  });

  it("should ignore the variable when the params give secure false", () => {
    const env = { SYNNAX_CA_CERT: "/missing.pem" };
    const resolved = resolveParams({ host: "rig", secure: false }, env, system);
    expect(resolved.secure).toBe(false);
    expect(resolved.caCert).toBeUndefined();
  });

  it("should fill the rest from Synnax Desktop", () => {
    const system: System = {
      readFile: (path) =>
        path === "~/ca.pem" ? PEM : JSON.stringify({ ...DESKTOP_FILE, secure: false }),
      run: () => ({ status: 0, stdout: "syk_desktop\n", stderr: "" }),
    };
    expect(resolveParams({}, { SYNNAX_CA_CERT: "~/ca.pem" }, system)).toEqual({
      host: "localhost",
      port: 9095,
      apiKey: "syk_desktop",
      secure: true,
      caCert: PEM,
    });
  });

  it.each<[string, Env, System | undefined, string]>([
    [
      "SYNNAX_SECURE is false",
      { SYNNAX_SECURE: "false", SYNNAX_CA_CERT: "~/ca.pem" },
      system,
      "SYNNAX_CA_CERT requires SYNNAX_SECURE; leave SYNNAX_SECURE out",
    ],
    [
      "the file does not exist",
      { SYNNAX_CA_CERT: "/missing.pem" },
      system,
      'SYNNAX_CA_CERT names "/missing.pem", which does not exist',
    ],
    [
      "the file is empty",
      { SYNNAX_CA_CERT: "/empty.pem" },
      system,
      'SYNNAX_CA_CERT names "/empty.pem", which is empty',
    ],
    [
      "the file is not PEM text",
      { SYNNAX_CA_CERT: "/ca.der" },
      system,
      'SYNNAX_CA_CERT names "/ca.der", which is not PEM text',
    ],
    [
      "the runtime has no file access",
      { SYNNAX_CA_CERT: "~/ca.pem" },
      undefined,
      'Cannot read SYNNAX_CA_CERT file "~/ca.pem": this runtime has no file access',
    ],
  ])("should reject when %s", (_, env, system, message) => {
    expect(() => resolveParams({ host: "rig" }, env, system)).toThrow(ValidationError);
    expect(() => resolveParams({ host: "rig" }, env, system)).toThrow(message);
  });

  it("should turn on TLS without reading the file when the CA is unsupported", () => {
    const deno: System = {
      readFile: vi.fn(() => PEM),
      run: () => null,
      caCertUnsupported: true,
    };
    const resolved = resolveParams(
      { host: "rig" },
      { SYNNAX_CA_CERT: "~/ca.pem" },
      deno,
    );
    expect(resolved.secure).toBe(true);
    expect(resolved.caCert).toBeUndefined();
    expect(deno.readFile).not.toHaveBeenCalled();
  });

  it("should reject SYNNAX_SECURE false when the CA is unsupported", () => {
    const deno: System = { ...system, caCertUnsupported: true };
    const env = { SYNNAX_SECURE: "false", SYNNAX_CA_CERT: "~/ca.pem" };
    expect(() => resolveParams({ host: "rig" }, env, deno)).toThrow(
      "SYNNAX_CA_CERT requires SYNNAX_SECURE",
    );
  });

  it("should name the file when Deno cannot read it", () => {
    const cause = notCapable();
    const system: System = {
      readFile: () => {
        throw cause;
      },
      run: () => null,
    };
    let error: unknown;
    try {
      resolveParams({ host: "rig" }, { SYNNAX_CA_CERT: "/ca.pem" }, system);
    } catch (e) {
      error = e;
    }
    expect(error).toBeInstanceOf(ValidationError);
    expect(error).toHaveProperty(
      "message",
      'Cannot read SYNNAX_CA_CERT file "/ca.pem": Requires env access',
    );
    expect(error).toHaveProperty("cause", cause);
  });

  describe.skipIf(process.platform === "win32")("with the real file system", () => {
    it("should read a file in the home directory", () => {
      const home = mkdtempSync(join(tmpdir(), "synnax-home-"));
      vi.stubEnv("HOME", home);
      try {
        writeFileSync(join(home, "ca.pem"), PEM);
        const env = { SYNNAX_CA_CERT: "~/ca.pem" };
        expect(resolveParams({ host: "rig" }, env, nodeSystem()).caCert).toEqual(PEM);
      } finally {
        vi.unstubAllEnvs();
        rmSync(home, { recursive: true });
      }
    });

    it("should return null for a file that is not there", () => {
      expect(nodeSystem()?.readFile(join(tmpdir(), "synnax-missing.pem"))).toBeNull();
    });
  });
});
