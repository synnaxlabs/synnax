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
import { TimeSpan } from "@synnaxlabs/x";
import { type DownloadEvent, type Update } from "@tauri-apps/plugin-updater";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useRef,
  useState,
} from "react";

import {
  type Available,
  type Blocked,
  find,
  STATUS_KEY_PREFIX,
  useTarget,
} from "@/platform/version/check";
import { CoreNotice } from "@/platform/version/CoreNotice";

const CHECK_INTERVAL = TimeSpan.seconds(30);

/** The result of the last update check. */
export type Check =
  | { variant: "loading" }
  | { variant: "error"; status: status.Status }
  | Blocked
  | { variant: "success"; available: Available | null };

/**
 * Downloads and installs an update that the Provider found, and sends the download
 * events to onEvent. Rejects when the checks started again after they found the update,
 * or when the download or the install fails.
 */
export type Install = (
  update: Update,
  onEvent: (event: DownloadEvent) => void,
) => Promise<void>;

interface ContextValue {
  check: Check;
  install: Install;
}

const Context = createContext<ContextValue>({
  check: { variant: "success", available: null },
  install: async () => {
    throw new Error("Version.Provider is missing");
  },
});
Context.displayName = "Version.Context";

/**
 * Runs the install of a downloaded update. It must call `install` exactly once and
 * reject when `install` rejects.
 */
export type InstallMiddleware = (install: () => Promise<void>) => Promise<void>;

const runInstall: InstallMiddleware = async (install) => await install();

export interface ProviderProps extends PropsWithChildren {
  /** Wraps the install of each update. */
  installMiddleware?: InstallMiddleware;
}

interface Installing {
  update: Update;
  /** True when the checks started again and left the update to the install. */
  released: boolean;
}

/**
 * Checks for an update now and every 30 seconds until it finds one, then adds a status.
 * The checks also stop when an insecure connection blocks a downgrade. A change of the
 * Core, its version, or the grant starts the checks again. The found update stays open
 * for {@link useCheck} until the checks start again or the Provider unmounts, and while
 * {@link useInstall} installs it. A status also tells a user who may take the Console
 * past the Core about a newer Core release.
 */
export const Provider = ({
  installMiddleware = runInstall,
  children,
}: ProviderProps): ReactElement => {
  const addStatus = Status.useAdder();
  const target = useTarget();
  const [check, setCheck] = useState<Check>({ variant: "loading" });
  const held = useRef<Update | null>(null);
  const installing = useRef<Installing | null>(null);
  useAsyncEffect(
    async (signal) => {
      setCheck({ variant: "loading" });
      // Not held.current: a newer run can set it before this cleanup runs.
      let found: Available | null = null;
      let timer: ReturnType<typeof setTimeout> | undefined;
      const run = async (): Promise<void> => {
        let next: Available | Blocked | null;
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
          if (next != null && next.variant !== "blocked") await next.update.close();
          return;
        }
        if (next?.variant === "blocked") {
          setCheck(next);
          return;
        }
        setCheck({ variant: "success", available: next });
        if (next == null) {
          timer = setTimeout(() => void run(), CHECK_INTERVAL.milliseconds);
          return;
        }
        found = next;
        held.current = next.update;
        addStatus({
          // A new check that finds the same version replaces the status.
          key: `${STATUS_KEY_PREFIX}-${next.update.version}`,
          variant: "info",
          message: "Update available",
        });
      };
      await run();
      return () => {
        clearTimeout(timer);
        if (found == null) return;
        held.current = null;
        if (installing.current?.update === found.update)
          installing.current.released = true;
        else found.update.close().catch(console.error);
      };
    },
    [target?.endpoint, target?.version, target?.latestGranted],
  );
  const install: Install = async (update, onEvent) => {
    if (held.current !== update)
      throw new Error(
        "The Core changed before the update started. Close this dialog and update again.",
      );
    const current: Installing = { update, released: false };
    installing.current = current;
    let failed = false;
    let error: unknown;
    try {
      await update.download(onEvent);
      await installMiddleware(async () => await update.install());
    } catch (e) {
      failed = true;
      error = e;
    }
    installing.current = null;
    if (current.released) await update.close().catch(console.error);
    if (failed)
      throw error instanceof Error ? error : new Error(String(error), { cause: error });
  };
  return (
    <Context value={{ check, install }}>
      <CoreNotice target={target} />
      {children}
    </Context>
  );
};

/**
 * @returns the result of the last check of the nearest Provider. Without one, no update
 * is available.
 */
export const useCheck = (): Check => use(Context).check;

/** @returns the function that installs an update that the nearest Provider found. */
export const useInstall = (): Install => use(Context).install;
