// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/pluto";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import {
  afterEach,
  describe,
  expect,
  it,
  type Mock,
  type MockInstance,
  vi,
} from "vitest";

import { Account } from "@/feature/account";
import { Session } from "@/session";
import { renderHookWithConsole, type TestStore } from "@/testutil";

const LINKED: Session.Account.SliceState = {
  version: 0,
  activation: "act",
  secret: "shh",
  email: "someone@example.com",
};

const DESKTOP_LICENSE: license.License = {
  jti: "0b4d1e6e-7a4b-4d2f-9c8e-1f2a3b4c5d6e",
  iat: 0,
  v: 1,
  org: "6e5d4c3b-2a1f-4e8c-9d2f-4b7a6e1d0c9b",
  ed: "d",
  fingerprints: ["aa"],
  fingerprintScheme: 1,
  n: 1,
  ch: 0,
};

const expiringIn = (span: TimeSpan): license.License => ({
  ...DESKTOP_LICENSE,
  exp: Math.floor(TimeStamp.now().add(span).seconds),
});

const infoOf = (lic: license.License | undefined): license.Info => ({
  state: lic == null ? "missing" : "ok",
  warning: undefined,
  fingerprint: ["aa"],
  license: lic,
});

interface Harness {
  retrieve: MockInstance<Client["license"]["retrieve"]>;
  activate: MockInstance<Client["license"]["activate"]>;
  store: TestStore;
  renew: Mock<(secret: string) => Promise<Account.RenewResult>>;
  statuses: () => Status.NotificationSpec[];
}

interface SetupOptions {
  license?: license.License;
  result?: Account.RenewResult;
  account?: Session.Account.SliceState;
}

const setup = async ({
  license: lic,
  result = { variant: "renewed", token: "x.y.z" },
  account = LINKED,
}: SetupOptions = {}): Promise<Harness> => {
  const client = createTestClient();
  const retrieve = vi.spyOn(client.license, "retrieve").mockResolvedValue(infoOf(lic));
  const activate = vi.spyOn(client.license, "activate").mockResolvedValue(infoOf(lic));
  const renew = vi.fn(async () => result);
  const { result: rendered, store } = await renderHookWithConsole(
    () => {
      Account.useRenew({ renew });
      return Status.useNotifications().statuses;
    },
    { client, preloadedState: { [Session.Account.SLICE_NAME]: account } },
  );
  return { retrieve, activate, store, renew, statuses: () => rendered.current };
};

describe("Account.useRenew", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should renew and apply the token once the license nears its expiry", async () => {
    const h = await setup({ license: expiringIn(TimeSpan.days(3)) });
    await waitFor(() => expect(h.activate).toHaveBeenCalledWith("x.y.z"));
    expect(h.renew).toHaveBeenCalledWith("shh");
  });

  it("should renew when no license applies", async () => {
    const h = await setup();
    await waitFor(() => expect(h.activate).toHaveBeenCalledWith("x.y.z"));
  });

  it("should leave a license alone while it has more than a week to run", async () => {
    const h = await setup({ license: expiringIn(TimeSpan.days(20)) });
    await waitFor(() => expect(h.retrieve).toHaveBeenCalled());
    expect(h.renew).not.toHaveBeenCalled();
  });

  it("should leave a perpetual license alone", async () => {
    const h = await setup({ license: DESKTOP_LICENSE });
    await waitFor(() => expect(h.retrieve).toHaveBeenCalled());
    expect(h.renew).not.toHaveBeenCalled();
  });

  it("should forget the account once the hub has unlinked the machine", async () => {
    const h = await setup({
      result: { variant: "unlinked", message: "This machine was unlinked." },
    });
    await waitFor(() =>
      expect(Session.Account.select(h.store.getState())).toEqual(
        Session.Account.ZERO_SLICE_STATE,
      ),
    );
    expect(h.activate).not.toHaveBeenCalled();
    expect(h.statuses().some((s) => s.message === "This machine was signed out")).toBe(
      true,
    );
  });

  it("should do nothing on a machine that is not linked", async () => {
    const h = await setup({ account: Session.Account.ZERO_SLICE_STATE });
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(h.retrieve).not.toHaveBeenCalled();
  });
});
