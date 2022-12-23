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
