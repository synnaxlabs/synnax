// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef } from "react";

import { type SynnaxProps } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { useAsyncWindowLifecycle, useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Status, useAsyncEffect, useSyncedRef } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { Child, Command, type EventEmitter } from "@tauri-apps/api/shell";
import { useDispatch } from "react-redux";

import { useSelectLocalState } from "@/cluster/selectors";
import { setLocalState, set, LOCAL_CLUSTER_KEY, setActive } from "@/cluster/slice";

import { testConnection } from "./testConnection";

// The name of the sidecar binary.
const BINARY_NAME = "bin/sy";
export const LOCAL_KEY = "local";

export const useLocalServer = (): void => {
  const win = useSelectWindowKey();

  const d = useDispatch();
  const { pid, state } = useSelectLocalState();
  const status = Status.useAggregator();
  const ref = useRef<EventEmitter<"data"> | null>(null);
  const pidRef = useSyncedRef(pid);

  const startLocalServer = async (): Promise<void> => {
    if (win !== Drift.MAIN_WINDOW) return;
    // The only case where we'll run into a stranded PID is if the user closes the
    // application or hard reloads the page. This means that we only need to kill
    // stranded pids on application load, so we don't pass the PID in as a dependency.
    if (pid !== 0) {
      console.log("Killing stranded local server", pid);
      await new Child(pid).kill();
    }

    const command = Command.sidecar(BINARY_NAME);
    const serverProcess = await command.spawn();

    d(setLocalState({ pid: serverProcess.pid, state: "starting" }));

    const handleLog = (v: string): void => {
      // All of our logs are JSON parseable.
      const { level, msg, error } = JSON.parse(v);

      const isInfo = level === "info";
      // This means the server has booted up.
      if (isInfo && msg === "starting server") {
        // Set the PID in local state so we can kill it later fi we need to.

        const props: SynnaxProps = {
          name: "Local",
          host: "localhost",
          port: 9090,
          username: "synnax",
          password: "seldon",
          secure: false,
        };

        // Test the connection to the local server.
        testConnection(props)
          .then(() => {
            d(
              set({
                key: LOCAL_CLUSTER_KEY,
                name: "Local",
                props,
              }),
            );
            d(setLocalState({ pid: serverProcess.pid, state: "running" }));
            d(setActive(LOCAL_KEY));
          })
          .catch(console.error);
      } else if (isInfo && msg === "shutdown successful") {
        // If the server has shut down, we'll set the PID to 0.
        d(setLocalState({ pid: 0, state: "stopped" }));
      }

      // If the server fails to boot up, we'll get a fatal error.
      if (level === "fatal")
        status({
          time: TimeStamp.now(),
          variant: "error",
          message: error,
          key: "local-server",
        });
    };
    ref.current = command.stderr.on("data", handleLog);
  };

  const stopLocalServer = useCallback(async (): Promise<void> => {
    if (pidRef.current === 0) return;
    // ref.current?.removeAllListeners("data");
    ref.current = null;
    d(setLocalState({ pid, state: "stopping" }));
    const serverProcess = new Child(pidRef.current);
    await serverProcess.write("stop\n");
    d(setActive(null));
  }, []);

  useAsyncEffect(async () => {
    if (win !== Drift.MAIN_WINDOW) return;
    if (state === "startCommanded") await startLocalServer();
    if (state === "stopCommanded") await stopLocalServer();
  }, [win, state]);

  useAsyncWindowLifecycle(async () => {
    await startLocalServer();
    return async () => await stopLocalServer();
  });
};
