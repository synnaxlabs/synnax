// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license, status } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { useEffect } from "react";

import { renew, type RenewResult } from "@/feature/account/portal";
import { Session } from "@/session";

/** How often a running app checks whether its license needs a renewal. */
export const CHECK_INTERVAL = TimeSpan.hours(6);

/** How close to its expiry a license is renewed. */
export const RENEW_WINDOW = TimeSpan.days(7);

/** Whether the license that applies still covers the machine for the renew window. */
const covered = (license: license.License | undefined, now: TimeStamp): boolean => {
  if (license == null) return false;
  if (license.exp == null) return true;
  return TimeStamp.seconds(license.exp).after(now.add(RENEW_WINDOW));
};

export interface RenewDeps {
  renew: (secret: string) => Promise<RenewResult>;
  interval: TimeSpan;
}

const DEFAULT_DEPS: RenewDeps = { renew, interval: CHECK_INTERVAL };

/**
 * Keeps a linked machine licensed: on launch and on an interval, renews through the
 * portal once the license is within a week of its expiry, or missing. A machine the
 * portal has unlinked forgets its account.
 */
export const useRenew = ({
  renew: renewToken = DEFAULT_DEPS.renew,
  interval = DEFAULT_DEPS.interval,
}: Partial<RenewDeps> = {}): void => {
  const client = Synnax.use();
  const { secret } = Session.Account.useSelect();
  const dispatch = Session.useDispatch();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    if (client == null || secret == null) return;
    const controller = new AbortController();
    const check = (): void =>
      handleError(async () => {
        const { license } = await client.license.retrieve();
        if (controller.signal.aborted || covered(license, TimeStamp.now())) return;
        const result = await renewToken(secret);
        if (controller.signal.aborted) return;
        if (result.variant === "unlinked") {
          dispatch(Session.Account.clear());
          addStatus(
            status.create({
              variant: "warning",
              message: "This machine was signed out",
              description: result.message,
            }),
          );
          return;
        }
        await client.license.activate(result.token);
      }, "Failed to renew the license");
    check();
    const timer = setInterval(check, interval.milliseconds);
    return () => {
      controller.abort();
      clearInterval(timer);
    };
  }, [client, secret, renewToken, interval, dispatch, addStatus, handleError]);
};
