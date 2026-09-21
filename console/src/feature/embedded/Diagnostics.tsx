// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/embedded/Diagnostics.css";

import { type status } from "@synnaxlabs/client";
import { Button, Flex, Icon, Status, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { Size, TimeStamp } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { type ReactElement, type ReactNode, useEffect, useRef, useState } from "react";

import { format } from "@/feature/embedded/log";
import { useStatus } from "@/feature/embedded/Provider";
import {
  type Diagnostics,
  exportDiagnostics,
  retrieveDiagnostics,
  retrieveLogTail,
  showData,
  showLogs,
  type Status as SupervisorStatus,
} from "@/feature/embedded/supervisor";
import { useRestart } from "@/feature/embedded/useRestart";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

const LOG_REFRESH_INTERVAL_MS = 2000;

const STATE_MESSAGES: Record<SupervisorStatus["state"], string> = {
  starting: "Starting",
  running: "Running",
  restarting: "Restarting",
  failed: "Stopped unexpectedly",
  stopping: "Stopping",
  stopped: "Stopped",
};

const STATE_VARIANTS: Record<SupervisorStatus["state"], status.Variant> = {
  starting: "loading",
  running: "success",
  restarting: "loading",
  failed: "error",
  stopping: "loading",
  stopped: "warning",
};

const formatSize = (bytes: number): string => {
  const size = Size.bytes(bytes);
  if (size.gigabytes >= 1) return `${size.gigabytes.toFixed(1)} GB`;
  if (size.megabytes >= 1) return `${size.megabytes.toFixed(1)} MB`;
  return `${size.kilobytes.toFixed(1)} kB`;
};

interface FieldProps {
  label: string;
  children: ReactNode;
}

const Field = ({ label, children }: FieldProps): ReactElement => (
  <Flex.Box x gap="medium" align="start">
    <Text.Text className={CSS.BE("diagnostics", "label")} color={9} weight={450}>
      {label}
    </Text.Text>
    <Text.Text className={CSS.BE("diagnostics", "value")}>{children}</Text.Text>
  </Flex.Box>
);

const useDiagnostics = (state: SupervisorStatus["state"]): Diagnostics | null => {
  const [diagnostics, setDiagnostics] = useState<Diagnostics | null>(null);
  const handleError = Status.useErrorHandler();
  useAsyncEffect(
    async (signal) => {
      try {
        const next = await retrieveDiagnostics();
        if (!signal.aborted) setDiagnostics(next);
      } catch (err) {
        if (!signal.aborted) handleError(err, "Failed to read the diagnostics");
      }
    },
    [state, handleError],
  );
  return diagnostics;
};

const useLog = (): string => {
  const [log, setLog] = useState("");
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    let stopped = false;
    const refresh = (): void =>
      handleError(async () => {
        const next = format(await retrieveLogTail());
        if (!stopped) setLog(next);
      }, "Failed to read the log");
    refresh();
    const interval = setInterval(refresh, LOG_REFRESH_INTERVAL_MS);
    return () => {
      stopped = true;
      clearInterval(interval);
    };
  }, [handleError]);
  return log;
};

const Log = (): ReactElement => {
  const log = useLog();
  const ref = useRef<HTMLPreElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (el != null) el.scrollTop = el.scrollHeight;
  }, [log]);
  return (
    <pre ref={ref} className={CSS.BE("diagnostics", "log")} aria-label="Log">
      {log}
    </pre>
  );
};

/**
 * Opens a dialog that shows the health of Synnax Desktop, the place and size of its
 * data, and the end of its log. From it, a person can restart the embedded Core and
 * export an archive for support.
 */
export const useDiagnosticsModal = Modals.create(() => {
  const status = useStatus();
  const diagnostics = useDiagnostics(status.state);
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const restart = useRestart();
  const handleExport = (): void =>
    handleError(async () => {
      const stamp = TimeStamp.now().toString("ISODate", "local");
      const path = await save({
        title: "Export diagnostics",
        defaultPath: `synnax-diagnostics-${stamp}.zip`,
        filters: [{ name: "Zip archive", extensions: ["zip"] }],
      });
      if (path == null) return;
      await exportDiagnostics(path);
      addStatus({ variant: "success", message: `Exported diagnostics to ${path}` });
    }, "Failed to export the diagnostics");
  const history = diagnostics?.history;
  return (
    <Modals.Frame className={CSS.B("diagnostics")}>
      <Modals.Header icon={<Icon.Hardware />}>Diagnostics</Modals.Header>
      <Modals.Body className={CSS.BE("diagnostics", "body")} gap="large">
        <Flex.Box y gap="small">
          <Status.Summary variant={STATE_VARIANTS[status.state]} level="h4">
            {STATE_MESSAGES[status.state]}
          </Status.Summary>
          {diagnostics != null && history != null && (
            <>
              <Field label="Version">{diagnostics.version}</Field>
              {history.readyAt != null && (
                <Field label="Running since">
                  {TimeStamp.milliseconds(history.readyAt).toString(
                    "dateTime",
                    "local",
                  )}
                </Field>
              )}
              <Field label="Starts">{history.starts} in this session</Field>
              {history.lastExit != null && (
                <Field label="Last problem">{history.lastExit}</Field>
              )}
              {status.state === "failed" && (
                <Field label="Reason">{status.message}</Field>
              )}
              <Field label="Data">
                {formatSize(diagnostics.dataSize)} in {diagnostics.dataDir}
              </Field>
            </>
          )}
        </Flex.Box>
        <Log />
      </Modals.Body>
      <Modals.Footer>
        <Flex.Box
          x
          gap="small"
          align="center"
          className={CSS.BE("diagnostics", "actions")}
        >
          <Button.Button
            variant="outlined"
            onClick={() => handleError(showData, "Failed to show the data folder")}
          >
            <Icon.Explore />
            Show data folder
          </Button.Button>
          <Button.Button
            variant="outlined"
            onClick={() => handleError(showLogs, "Failed to show the logs")}
          >
            <Icon.Log />
            Show logs
          </Button.Button>
          <Button.Button variant="outlined" onClick={handleExport}>
            <Icon.Export />
            Export
          </Button.Button>
          <Button.Button variant="filled" onClick={restart}>
            <Icon.Refresh />
            Restart
          </Button.Button>
        </Flex.Box>
      </Modals.Footer>
    </Modals.Frame>
  );
});
