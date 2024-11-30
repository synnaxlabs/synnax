import { Drift } from "@synnaxlabs/drift";
import { useAsyncWindowLifecycle, useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Status, useAsyncEffect, useSyncedRef } from "@synnaxlabs/pluto";
import {
  Child,
  Command,
  type CommandEvents,
  type EventEmitter,
} from "@tauri-apps/plugin-shell";
import { useCallback, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { parseLogMessage, useLogsContext } from "@/cluster/embedded/LogsProvider";
import { useSelectEmbeddedState } from "@/cluster/selectors";
import {
  LOCAL,
  LOCAL_CLUSTER_KEY,
  LOCAL_PROPS,
  set,
  setActive,
  setLocalState,
} from "@/cluster/slice";
import { testConnection } from "@/cluster/testConnection";

// The name of the sidecar binary.
const BINARY_NAME = "bin/sy";

export const use = (): void => {
  const win = useSelectWindowKey();
  const d = useDispatch();
  const { pid, command } = useSelectEmbeddedState();
  const status = Status.useAggregator();
  const pidRef = useSyncedRef(pid);
  const { addLog } = useLogsContext();
  const commandRef = useRef<EventEmitter<CommandEvents> | null>(null);

  const start = async (): Promise<void> => {
    if (win !== Drift.MAIN_WINDOW) return;
    if (commandRef.current != null) {
      commandRef.current.removeAllListeners();
      commandRef.current = null;
    }
    // The only case where we'll run into a stranded PID is if the user closes the
    // application or hard reloads the page. This means that we only need to kill
    // stranded PIDs on application load, so we don't pass the PID in as a dependency.
    if (pid !== 0) {
      console.log("Killing stranded local server", pid);
      const child = new Child(pid);
      await child.kill();
    }

    const command = Command.sidecar(BINARY_NAME, [
      "start",
      "-vmi",
      // "-d",
      // "/Users/emilianobonilla/Desktop/synnaxlabs/synnax/synnax-data",
    ]);
    commandRef.current = command;

    const handleLog = (v: string): void => {
      const log = parseLogMessage(v);
      if (log == null) return;
      addLog(log);
      const { level, msg, error } = log;

      const isInfo = level === "info";
      // This means the server has booted up.
      if (isInfo && msg === "starting server")
        // Test the connection to the local server.
        testConnection(LOCAL_PROPS)
          .then(() => {
            d(set(LOCAL));
            d(setLocalState({ pid: serverProcess.pid, status: "running" }));
            d(setActive(LOCAL_CLUSTER_KEY));
          })
          .catch(console.error);
      else if (isInfo && msg === "shutdown successful")
        // If the server has shut down, we'll set the PID to 0.
        d(setLocalState({ pid: 0, status: "stopped" }));

      // If the server fails to boot up, we'll get a fatal error.
      if (level === "fatal")
        status({
          variant: "error",
          message: error,
          key: "local-server",
        });
    };

    const handleClose = (): void => {
      d(setLocalState({ pid: 0, status: "stopped" }));
    };

    command.stderr.on("data", handleLog);
    command.on("close", handleClose);
    const serverProcess = await command.spawn();

    d(setLocalState({ pid: serverProcess.pid, status: "starting" }));
  };

  const stop = useCallback(async (): Promise<void> => {
    if (pidRef.current === 0) return;
    d(setLocalState({ pid, status: "stopping" }));
    d(setActive(null));
    const serverProcess = new Child(pidRef.current);
    await serverProcess.write("stop\n");
  }, []);

  const kill = useCallback(async (): Promise<void> => {
    if (pidRef.current === 0) return;
    const serverProcess = new Child(pidRef.current);
    serverProcess.kill();
    d(setLocalState({ pid: 0, status: "stopped" }));
    d(setActive(null));
  }, []);

  useAsyncEffect(async () => {
    if (win !== Drift.MAIN_WINDOW) return;
    if (command === "start") return await start();
    if (command === "stop") return await stop();
    if (command === "kill") return await kill();
  }, [win, command]);

  useAsyncWindowLifecycle(async () => async () => {
    d(setActive(null));
    await stop();
  });

  useEffect(() => {
    d(setLocalState({ pid: 0, status: "stopped" }));
  }, []);
};
