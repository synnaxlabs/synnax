// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { id, TimeSpan } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useState,
} from "react";

import {
  type Available,
  find,
  STATUS_KEY_PREFIX,
  useTarget,
} from "@/platform/version/check";

const CHECK_INTERVAL = TimeSpan.seconds(30);

/** The result of the last update check. */
export type Check =
  | { variant: "loading" }
  | { variant: "error"; status: status.Status }
  | { variant: "success"; available: Available | null };

const Context = createContext<Check>({ variant: "success", available: null });
Context.displayName = "Version.Context";

/**
 * Checks for an update now and every 30 seconds until it finds one, then adds a status.
 * A change of the Core, its version, or the grant starts the checks again. The found
 * update stays open for {@link useCheck} until the checks start again or the Provider
 * unmounts.
 */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const addStatus = Status.useAdder();
  const target = useTarget();
  const [check, setCheck] = useState<Check>({ variant: "loading" });
  useAsyncEffect(
    async (signal) => {
      setCheck({ variant: "loading" });
      let found: Available | null = null;
      let timer: ReturnType<typeof setTimeout> | undefined;
      const run = async (): Promise<void> => {
        let next: Available | null;
        try {
          next = await find(target);
        } catch (error) {
          if (signal.aborted) return;
          setCheck({
            variant: "error",
            status: status.fromException(error, "Failed to check for updates"),
          });
          timer = setTimeout(() => void run(), CHECK_INTERVAL.milliseconds);
          return;
        }
        if (signal.aborted) {
          await next?.update.close();
          return;
        }
        setCheck({ variant: "success", available: next });
        if (next == null) {
          timer = setTimeout(() => void run(), CHECK_INTERVAL.milliseconds);
          return;
        }
        found = next;
        addStatus({
          key: `${STATUS_KEY_PREFIX}-${id.create()}`,
          variant: "info",
          message: "Update available",
        });
      };
      await run();
      return () => {
        clearTimeout(timer);
        found?.update.close().catch(console.error);
      };
    },
    [target?.endpoint, target?.version, target?.latestGranted],
  );
  return <Context value={check}>{children}</Context>;
};

/**
 * @returns the result of the last check of the nearest Provider. Without one, no update
 * is available.
 */
export const useCheck = (): Check => use(Context);
