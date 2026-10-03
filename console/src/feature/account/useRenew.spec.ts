// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it, type Mock, vi } from "vitest";

import { Account } from "@/feature/account";
import { Session } from "@/session";
import { renderHookWithConsole, type TestStore } from "@/testutil";

const LINKED: Session.Account.SliceState = {
  version: 0,
  link: { secret: "shh", email: "someone@example.com", user: "user_a" },
};

const DESKTOP_LICENSE: license.License = {
  jti: "0b4d1e6e-7a4b-4d2f-9c8e-1f2a3b4c5d6e",
  iat: 0,
  claimsVersion: 1,
  organization: "6e5d4c3b-2a1f-4e8c-9d2f-4b7a6e1d0c9b",
  edition: "d",
  fingerprints: ["aa"],
  fingerprintScheme: 1,
  machines: 1,
  channels: 0,
  required: [],
};

const expiringIn = (span: TimeSpan): license.License => ({
  ...DESKTOP_LICENSE,
  exp: Math.floor(TimeStamp.now().add(span).seconds),
});

const infoOf = (lic: license.License | undefined): license.Info => ({
  state: lic == null ? "missing" : "ok",
  warning: "",
  fingerprint: ["aa"],
  license: lic,
});

interface Harness {
  retrieve: Mock<Account.RenewDeps["license"]["retrieve"]>;
  activate: Mock<Account.RenewDeps["license"]["activate"]>;
  deactivate: Mock<Account.RenewDeps["license"]["deactivate"]>;
  store: TestStore;
  renew: Mock<(secret: string) => Promise<Account.RenewResult>>;
  statuses: () => Status.NotificationSpec[];
}

interface SetupOptions {
  license?: license.License;
  result?: Account.RenewResult;
  account?: Session.Account.SliceState;
  interval?: TimeSpan;
  deactivateError?: Error;
  renewError?: Error;
}

const setup = async ({
  license: lic,
  result = { variant: "renewed", key: "x.y.z" },
  account = LINKED,
  interval,
  deactivateError,
  renewError,
}: SetupOptions = {}): Promise<Harness> => {
  const retrieve = vi.fn(async () => infoOf(lic));
  const activate = vi.fn(async () => infoOf(lic));
  const deactivate = vi.fn(async () => {
    if (deactivateError != null) throw deactivateError;
    return infoOf(undefined);
  });
  const renew = vi.fn(async () => {
    if (renewError != null) throw renewError;
    return result;
  });
  const deps: Partial<Account.RenewDeps> = {
    renew,
    interval,
    license: { retrieve, activate, deactivate },
  };
  const { result: rendered, store } = await renderHookWithConsole(
    () => {
      Account.useRenew(deps);
      return Status.useNotifications().statuses;
    },
    {
      client: createTestClient(),
      preloadedState: { [Session.Account.SLICE_NAME]: account },
    },
  );
  return {
    retrieve,
    activate,
    deactivate,
    store,
    renew,
    statuses: () => rendered.current,
  };
};

describe("Account.useRenew", () => {
  it("should renew and apply the license key while the license has weeks to run", async () => {
    const h = await setup({ license: expiringIn(TimeSpan.days(20)) });
    await waitFor(() => expect(h.activate).toHaveBeenCalledWith("x.y.z"));
    expect(h.renew).toHaveBeenCalledWith("shh");
  });

  it("should renew when no license applies", async () => {
    const h = await setup();
    await waitFor(() => expect(h.activate).toHaveBeenCalledWith("x.y.z"));
  });

  it("should renew again on each interval", async () => {
    const h = await setup({
      license: expiringIn(TimeSpan.days(20)),
      interval: TimeSpan.milliseconds(20),
    });
    await waitFor(() => expect(h.renew.mock.calls.length).toBeGreaterThan(1));
  });

  it("should stay quiet when the hub is unreachable and weeks remain", async () => {
    const h = await setup({
      license: expiringIn(TimeSpan.days(20)),
      renewError: new Error("Failed to fetch"),
      interval: TimeSpan.milliseconds(20),
    });
    await waitFor(() => expect(h.renew.mock.calls.length).toBeGreaterThan(1));
    expect(h.statuses()).toEqual([]);
  });

  it("should show a failed renewal within a week of the expiry", async () => {
    const h = await setup({
      license: expiringIn(TimeSpan.days(3)),
      renewError: new Error("Failed to fetch"),
    });
    await waitFor(() =>
      expect(
        h.statuses().some((s) => s.message === "Failed to renew the license"),
      ).toBe(true),
    );
  });

  it("should leave a perpetual license alone", async () => {
    const h = await setup({ license: DESKTOP_LICENSE });
    await waitFor(() => expect(h.retrieve).toHaveBeenCalled());
    expect(h.renew).not.toHaveBeenCalled();
  });

  it("should remove the license key once the hub has unlinked the machine", async () => {
    const lic = expiringIn(TimeSpan.days(20));
    const h = await setup({
      license: lic,
      result: { variant: "unlinked", message: "This machine was logged out." },
    });
    await waitFor(() => expect(h.deactivate).toHaveBeenCalledWith(lic.jti));
  });

  it("should keep the account when the license key cannot be removed", async () => {
    const h = await setup({
      license: expiringIn(TimeSpan.days(20)),
      result: { variant: "unlinked", message: "This machine was logged out." },
      deactivateError: new Error("Core unavailable"),
    });
    await waitFor(() => expect(h.deactivate).toHaveBeenCalled());
    expect(Session.Account.selectSliceState(h.store.getState())).toEqual(LINKED);
  });

  it("should leave an enterprise license when the hub unlinks the machine", async () => {
    const h = await setup({
      license: { ...expiringIn(TimeSpan.days(20)), edition: "e" },
      result: { variant: "unlinked", message: "This machine was logged out." },
    });
    await waitFor(() =>
      expect(Session.Account.selectSliceState(h.store.getState())).toEqual(
        Session.Account.ZERO_SLICE_STATE,
      ),
    );
    expect(h.deactivate).not.toHaveBeenCalled();
  });

  it("should forget the account once the hub has unlinked the machine", async () => {
    const h = await setup({
      result: { variant: "unlinked", message: "This machine was logged out." },
    });
    await waitFor(() =>
      expect(Session.Account.selectSliceState(h.store.getState())).toEqual(
        Session.Account.ZERO_SLICE_STATE,
      ),
    );
    expect(h.activate).not.toHaveBeenCalled();
    expect(h.statuses().some((s) => s.message === "This machine was logged out")).toBe(
      true,
    );
  });

  it("should do nothing on a machine that is not linked", async () => {
    const h = await setup({ account: Session.Account.ZERO_SLICE_STATE });
    expect(h.retrieve).not.toHaveBeenCalled();
  });
});
