// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SynnaxProps } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { Child, Command } from "@tauri-apps/api/shell";
import { useDispatch } from "react-redux";

import { useSelectLocalState } from "@/cluster/selectors";
import { setLocalState, set } from "@/cluster/slice";

import { testConnection } from "./testConnection";

// The name of the sidecar binary.
const BINARY_NAME = "bin/sy";
export const LOCAL_KEY = "local";

export const useLocalServer = (): void => {
  const win = useSelectWindowKey();

  const d = useDispatch();
  const { pid } = useSelectLocalState();
  const status = Status.useAggregator();

  useAsyncEffect(async () => {
    if (win !== Drift.MAIN_WINDOW) return;

    // The only case where we'll run into a stranded PID is if the user closes the
    // application or hard reloads the page. This means that we only need to kill
    // stranded pids on application load, so we don't pass the PID in as a dependency.
    if (pid !== 0) {
      await new Child(pid).kill();
    }

    const command = Command.sidecar(BINARY_NAME);
    const serverProcess = await command.spawn();

    const handleLog = async (v: string): Promise<void> => {
      // All of our logs are JSON parseable.
      const { level, msg, error } = JSON.parse(v);

      // This means the server has booted up.
      if (level === "info" && msg === "starting server") {
        // Set the PID in local state so we can kill it later fi we need to.
        d(setLocalState({ pid: serverProcess.pid }));

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
          .then(() =>
            d(
              set({
                key: "local",
                name: "Local",
                props,
              }),
            ),
          )
          .catch(console.error);
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
    const handleClose = (): void => {
      console.log("CLOSE");
      d(setLocalState({ pid: 0 }));
    };

    const dataListener = command.stderr.on("data", handleLog);

    command.on("close", handleClose);

    return () => {
      void serverProcess.kill();
      dataListener.off("data", handleLog);
      command.off("close", handleClose);
    };
  }, [win]);
};
