// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license, status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { errors, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { useEffect } from "react";

import { renew, type RenewResult } from "@/feature/account/handoff";
import { Session } from "@/session";

/**
 * How often a running app renews its license. It bounds how long a revoked machine keeps
 * its license while online.
 */
const CHECK_INTERVAL = TimeSpan.hours(6);

/** How close to its expiry a license must be before a failed renewal is shown. */
const WARNING_WINDOW = TimeSpan.days(7);

/** Whether the license still has more than the warning window to run. */
const lasting = (license: license.License | undefined): boolean =>
  license?.exp != null &&
  TimeStamp.seconds(license.exp).after(TimeStamp.now().add(WARNING_WINDOW));

export interface RenewDeps {
  renew: (secret: string) => Promise<RenewResult>;
  interval: TimeSpan;
  /** The Core's license API. Defaults to the connected client's. */
  license: Pick<license.Client, "retrieve" | "activate" | "deactivate">;
}

/**
 * Keeps a linked machine licensed: on launch and on an interval, renews through the hub
 * unless a perpetual license applies. A failed renewal shows only within a week of the
 * license's expiry. A machine the hub has unlinked removes its Desktop license key and
 * forgets its account.
 */
export const useRenew = ({
  renew: renewKey = renew,
  interval = CHECK_INTERVAL,
  license: injected,
}: Partial<RenewDeps> = {}): void => {
  const client = Synnax.use();
  const api = injected ?? client?.license;
  const secret = Session.Account.useSelectSecret();
  const dispatch = Session.useDispatch();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    if (api == null || secret == null) return;
    const controller = new AbortController();
    const check = (): void =>
      handleError(async () => {
        const { license } = await api.retrieve();
        if (controller.signal.aborted || (license != null && license.exp == null))
          return;
        let result: RenewResult;
        try {
          result = await renewKey(secret);
        } catch (e) {
          if (!lasting(license)) throw errors.fromUnknown(e);
          console.error("failed to renew the license", e);
          return;
        }
        if (controller.signal.aborted) return;
        if (result.variant === "unlinked") {
          if (license?.edition === "d") await api.deactivate(license.jti);
          dispatch(Session.Account.clear());
          addStatus(
            status.create({
              variant: "warning",
              message: "This machine was logged out",
              description: result.message,
            }),
          );
          return;
        }
        await api.activate(result.key);
      }, "Failed to renew the license");
    check();
    const timer = setInterval(check, interval.milliseconds);
    return () => {
      controller.abort();
      clearInterval(timer);
    };
  }, [api, secret, renewKey, interval, dispatch, addStatus, handleError]);
};
