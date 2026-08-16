// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/errors/Overlay.css";

import {
  Button,
  CSS as PCSS,
  Errors,
  Flex,
  OS,
  Synnax,
  Theming,
} from "@synnaxlabs/pluto";
import { type record, type runtime } from "@synnaxlabs/x";
import { getVersion } from "@tauri-apps/api/app";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { Shell } from "@/platform/shell";
import { Session } from "@/session";

export interface OverlayProps extends PropsWithChildren {}

interface ExtraErrorInfo extends record.Unknown {
  consoleVersion: string;
  coreVersion: string;
}

const useExtraErrorInfo = (): ExtraErrorInfo => {
  // These hooks must be called unconditionally per React rules.
  // If they throw, the error bubbles to OverlayWithoutStore which is fine.
  // We use optional chaining when building extraInfo to handle undefined values.
  const consoleVersion = Session.Version.use();
  const connectionStatus = Synnax.useConnectionStatus();
  const extraInfo: ExtraErrorInfo = {
    consoleVersion: "unknown",
    coreVersion: "unknown",
  };
  if (consoleVersion != null) extraInfo.consoleVersion = consoleVersion;
  if (connectionStatus?.details.nodeVersion != null)
    extraInfo.coreVersion = connectionStatus.details.nodeVersion;
  return extraInfo;
};

const FallbackRenderWithStore = ({ error }: Errors.FallbackProps): ReactElement => {
  const dispatch = Session.useDispatch();
  const extraInfo = useExtraErrorInfo();

  const handleTryAgain = useCallback((): void => {
    dispatch(Session.Persist.revertState());
  }, [dispatch]);
  const handleClear = useCallback((): void => {
    dispatch(Session.Persist.clearState());
  }, [dispatch]);

  return (
    <FallBackRenderContent<ExtraErrorInfo>
      onClear={handleClear}
      onTryAgain={handleTryAgain}
      error={error}
      extraInfo={extraInfo}
    />
  );
};

const FallbackRenderWithoutStore = ({ error }: Errors.FallbackProps): ReactElement => {
  const [consoleVersion, setConsoleVersion] = useState<string | undefined>();

  useEffect(() => {
    if (Session.Runtime.ENGINE !== "tauri") return;
    void getVersion().then(setConsoleVersion);
  }, []);

  const extraInfo: ExtraErrorInfo = {
    consoleVersion: consoleVersion ?? "unknown",
    coreVersion: "unknown",
  };

  return (
    <FallBackRenderContent
      onClear={Session.Persist.hardClearAndReload}
      error={error}
      extraInfo={extraInfo}
    />
  );
};

// Tauri-direct rather than Window.Controls: the store-less overlay catches failures
// from above the store, where that component's Drift selectors are unavailable.
const windowAction = (action: "close" | "minimize" | "maximize") => (): void => {
  if (Session.Runtime.ENGINE !== "tauri") return;
  void getCurrentWindow()[action]();
};

const handleClose = windowAction("close");
const handleMinimize = windowAction("minimize");
const handleMaximize = windowAction("maximize");

interface WindowControlsProps {
  os: runtime.OS;
}

const WindowControls = ({ os }: WindowControlsProps): ReactElement | null => {
  if (Session.Runtime.ENGINE !== "tauri") return null;
  if (os !== "macOS" && os !== "Windows") return null;
  return (
    <Shell.Islands justify={os === "macOS" ? "start" : "end"}>
      <Shell.Island>
        <OS.Controls
          forceOS={os}
          onClose={handleClose}
          onMinimize={handleMinimize}
          onMaximize={handleMaximize}
        />
      </Shell.Island>
    </Shell.Islands>
  );
};

interface FallbackRenderContentProps<
  ExtraInfo extends record.Unknown = record.Unknown,
> {
  error: Error;
  onTryAgain?: () => void;
  onClear: () => void;
  extraInfo?: ExtraInfo;
}

const FallBackRenderContent = <ExtraInfo extends record.Unknown = record.Unknown>({
  onTryAgain,
  onClear,
  error,
  extraInfo,
}: FallbackRenderContentProps<ExtraInfo>): ReactElement => {
  const os = OS.use();
  useEffect(() => {
    try {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      const theme = mediaQuery.matches ? Theming.SYNNAX_DARK : Theming.SYNNAX_LIGHT;
      PCSS.applyVars(
        document.documentElement,
        Theming.toCSSVars(Theming.themeZ.parse(theme)),
      );
    } catch (e) {
      console.error(e);
    }
    if (Session.Runtime.ENGINE === "tauri") void getCurrentWindow().show();
  }, []);
  const resetErrorBoundary = useCallback((): void => {
    onTryAgain?.();
  }, [onTryAgain]);

  return (
    <Flex.Box
      y
      full
      empty
      background={1}
      data-tauri-drag-region
      className={CSS.B("error-overlay")}
    >
      <WindowControls os={os} />
      <Errors.Fallback
        error={error}
        resetErrorBoundary={resetErrorBoundary}
        extraInfo={extraInfo}
      >
        <Flex.Box x>
          <Button.Button
            onClick={onClear}
            tooltip={`Will clear all stored data in the Console and reload the application.
              This should only be done if the standard reload does not fix the issue.`}
            tooltipLocation="bottom"
          >
            Clear storage and reload Console
          </Button.Button>
          {onTryAgain != null && (
            <Button.Button variant="filled" onClick={onTryAgain}>
              Reload Console
            </Button.Button>
          )}
        </Flex.Box>
      </Errors.Fallback>
    </Flex.Box>
  );
};

export const OverlayWithStore = (props: OverlayProps): ReactElement => (
  <Errors.Boundary {...props} FallbackComponent={FallbackRenderWithStore} />
);

export const OverlayWithoutStore = (props: OverlayProps): ReactElement => (
  <Errors.Boundary {...props} FallbackComponent={FallbackRenderWithoutStore} />
);
