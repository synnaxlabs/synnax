// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/embedded/useDiagnosticsModal.css";

import { type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Description } from "@synnaxlabs/lyra/description";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Size, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import {
  type ReactElement,
  type ReactNode,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { type Entry, parse, toText } from "@/feature/embedded/log";
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
import { useReset } from "@/feature/embedded/useReset";
import { useRestart } from "@/feature/embedded/useRestart";
import { useReveal } from "@/feature/embedded/useReveal";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

const LOG_REFRESH_INTERVAL = TimeSpan.seconds(2);

// How near its end a scrolled log still follows new lines, for fractional pixels.
const END_SLACK_PX = 8;

const LEVEL_MODIFIERS: Record<string, string> = {
  debug: "debug",
  warn: "warn",
  error: "error",
  dpanic: "error",
  panic: "error",
  fatal: "error",
};

const STATE_MESSAGES: Record<SupervisorStatus["state"], string> = {
  starting: "Starting",
  running: "Running",
  restarting: "Restarting",
  waiting: "Waiting for data folder",
  failed: "Stopped unexpectedly",
  stopping: "Stopping",
  exiting: "Stopping to exit",
  stopped: "Stopped",
};

const STATE_VARIANTS: Record<SupervisorStatus["state"], status.Variant> = {
  starting: "loading",
  running: "success",
  restarting: "loading",
  waiting: "loading",
  failed: "error",
  stopping: "loading",
  exiting: "loading",
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
  <Description.Item>
    <Description.Label>{label}</Description.Label>
    <Description.Value>{children}</Description.Value>
  </Description.Item>
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
        if (!signal.aborted) handleError(err, "Failed to read diagnostics");
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
        const next = await retrieveLogTail();
        if (!stopped) setLog(next);
      }, "Failed to read log");
    refresh();
    const interval = setInterval(refresh, LOG_REFRESH_INTERVAL.milliseconds);
    return () => {
      stopped = true;
      clearInterval(interval);
    };
  }, [handleError]);
  return log;
};

const Line = ({ level, time, logger, text }: Entry): ReactElement => {
  const modifier = level == null ? undefined : LEVEL_MODIFIERS[level];
  return (
    <span
      className={CSS.cls(
        CSS.BE("diagnostics", "line"),
        modifier != null && CSS.BEM("diagnostics", "line", modifier),
      )}
    >
      {time != null && (
        <span className={CSS.BE("diagnostics", "time")}>
          {time.toString("time", "local")}{" "}
        </span>
      )}
      {level != null && (
        <span className={CSS.BE("diagnostics", "level")}>{level.toUpperCase()} </span>
      )}
      {logger !== "" && (
        <span className={CSS.BE("diagnostics", "logger")}>{logger} </span>
      )}
      {text}
    </span>
  );
};

const isAtEnd = (el: HTMLElement): boolean =>
  el.scrollHeight - el.scrollTop - el.clientHeight <= END_SLACK_PX;

const holdsSelection = (el: HTMLElement): boolean => {
  const selection = getSelection();
  if (selection == null || selection.isCollapsed) return false;
  return selection.getRangeAt(0).intersectsNode(el);
};

interface LogProps {
  log: string;
}

/**
 * Follows new lines while the view sits at the end of the log. A scroll up or a text
 * selection holds the lines still until the person comes back to the end.
 */
const Log = ({ log }: LogProps): ReactElement => {
  const ref = useRef<HTMLPreElement>(null);
  const [held, setHeld] = useState<string | null>(null);
  const shown = held ?? log;
  // A hold keeps the lines on screen, which can be newer than a handler's closure.
  const shownRef = useRef(shown);
  useLayoutEffect(() => {
    shownRef.current = shown;
    const el = ref.current;
    if (held == null && el != null) el.scrollTop = el.scrollHeight;
  }, [shown, held]);
  const hold = (): void => setHeld((prev) => prev ?? shownRef.current);
  useEffect(() => {
    const handleSelectionChange = (): void => {
      const el = ref.current;
      if (el == null) return;
      if (holdsSelection(el)) hold();
      else if (isAtEnd(el)) setHeld(null);
    };
    document.addEventListener("selectionchange", handleSelectionChange);
    return () => document.removeEventListener("selectionchange", handleSelectionChange);
  }, []);
  const handleScroll = (): void => {
    const el = ref.current;
    if (el == null) return;
    if (isAtEnd(el) && !holdsSelection(el)) setHeld(null);
    else hold();
  };
  return (
    <div className={CSS.BE("diagnostics", "log-frame")}>
      <pre
        ref={ref}
        className={CSS.BE("diagnostics", "log")}
        aria-label="Log"
        onScroll={handleScroll}
      >
        {parse(shown).map((entry, i) => (
          <Line key={i} {...entry} />
        ))}
      </pre>
      {held != null && (
        <Button.Button
          className={CSS.BE("diagnostics", "latest")}
          variant="filled"
          size="small"
          onClick={() => {
            getSelection()?.removeAllRanges();
            setHeld(null);
          }}
        >
          <Icon.Arrow.Down />
          Jump to latest
        </Button.Button>
      )}
    </div>
  );
};

const DiagnosticsModal = () => {
  const status = useStatus();
  const diagnostics = useDiagnostics(status.state);
  const log = useLog();
  const getCopyText = (): string =>
    [
      "Synnax Desktop diagnostics",
      ...(diagnostics == null ? [] : [`Version: ${diagnostics.version}`]),
      `State: ${STATE_MESSAGES[status.state]}`,
      `Copied: ${TimeStamp.now().toString("ISO", "UTC")}`,
      "",
      toText(parse(log)),
    ].join("\n");
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const restart = useRestart();
  const reset = useReset();
  const revealData = useReveal(showData, "Failed to show data folder");
  const revealLogs = useReveal(showLogs, "Failed to show logs");
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
      addStatus({
        variant: "success",
        message: "Exported diagnostics",
        description: `Saved to ${path}.`,
      });
    }, "Failed to export diagnostics");
  const history = diagnostics?.history;
  return (
    <Modal.Frame className={CSS.B("diagnostics")}>
      <Modal.Header icon={<Icon.Hardware />}>Diagnostics</Modal.Header>
      <Modal.Body className={CSS.BE("diagnostics", "body")} gap="large">
        <Flex.Box y gap="small">
          <Status.Summary variant={STATE_VARIANTS[status.state]} level="h4">
            {STATE_MESSAGES[status.state]}
          </Status.Summary>
          {diagnostics != null && history != null && (
            <Description.List>
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
                <Field label="Last problem">{history.lastExit.message}</Field>
              )}
              {status.state === "failed" && (
                <Field label="Reason">{status.message}</Field>
              )}
              <Field label="Data">
                {formatSize(diagnostics.dataSize)} in {diagnostics.dataDir}
              </Field>
            </Description.List>
          )}
        </Flex.Box>
        <Log log={log} />
      </Modal.Body>
      <Modal.Footer>
        <Flex.Box
          x
          gap="small"
          align="center"
          className={CSS.BE("diagnostics", "actions")}
        >
          {Session.Runtime.isMainWindow() && (
            <Button.Button
              className={CSS.BE("diagnostics", "erase")}
              variant="text"
              status="error"
              onClick={reset}
            >
              <Icon.Delete />
              Erase all data
            </Button.Button>
          )}
          <Button.Button variant="outlined" onClick={revealData}>
            <Icon.Explore />
            Show data folder
          </Button.Button>
          <Button.Button variant="outlined" onClick={revealLogs}>
            <Icon.Log />
            Show logs
          </Button.Button>
          <Button.Copy
            variant="outlined"
            text={getCopyText}
            tooltip="Copy the recent log"
          >
            Copy logs
          </Button.Copy>
          <Button.Button variant="outlined" onClick={handleExport}>
            <Icon.Export />
            Export
          </Button.Button>
          <Button.Button variant="filled" onClick={restart}>
            <Icon.Refresh />
            Restart
          </Button.Button>
        </Flex.Box>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a dialog that shows the health of Synnax Desktop, the place and size of its
 * data, and the end of its log. From it, a person can restart the embedded Core and
 * export an archive for support.
 */
export const useDiagnosticsModal = Modals.create(DiagnosticsModal);
