// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/errors/Overlay.css";

import { Logo } from "@synnaxlabs/media";
import {
  Button,
  CSS as PCSS,
  Errors,
  Flex,
  Nav,
  OS,
  Synnax,
  Theming,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { getVersion } from "@tauri-apps/api/app";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Persist } from "@/persist";
import { CLEAR_STATE, REVERT_STATE } from "@/persist/state";
import { Runtime } from "@/runtime";
import { useSelectVersion } from "@/version/selectors";

export interface OverlayProps extends PropsWithChildren {}

interface ExtraErrorInfo extends record.Unknown {
  consoleVersion: string;
  coreVersion: string;
}

const useExtraErrorInfo = (): ExtraErrorInfo => {
  // These hooks must be called unconditionally per React rules.
  // If they throw, the error bubbles to OverlayWithoutStore which is fine.
  // We use optional chaining when building extraInfo to handle undefined values.
  const consoleVersion = useSelectVersion();
  const connectionState = Synnax.useConnectionState();

  const extraInfo: ExtraErrorInfo = {
    consoleVersion: "unknown",
    coreVersion: "unknown",
  };
  if (consoleVersion != null) extraInfo.consoleVersion = consoleVersion;
  if (connectionState?.nodeVersion != null)
    extraInfo.coreVersion = connectionState.nodeVersion;
  return extraInfo;
};

const FallbackRenderWithStore = ({ error }: Errors.FallbackProps): ReactElement => {
  const dispatch = useDispatch();
  const extraInfo = useExtraErrorInfo();

  const handleTryAgain = useCallback((): void => {
    dispatch(REVERT_STATE);
  }, [dispatch]);
  const handleClear = useCallback((): void => {
    dispatch(CLEAR_STATE);
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
    if (Runtime.ENGINE !== "tauri") return;
    void getVersion().then(setConsoleVersion);
  }, []);

  const extraInfo: ExtraErrorInfo = {
    consoleVersion: consoleVersion ?? "unknown",
    coreVersion: "unknown",
  };

  return (
    <FallBackRenderContent
      onClear={Persist.hardClearAndReload}
      error={error}
      extraInfo={extraInfo}
    />
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
    if (Runtime.ENGINE === "tauri") void getCurrentWindow().show();
  }, []);
  const resetErrorBoundary = useCallback((): void => {
    onTryAgain?.();
  }, [onTryAgain]);

  return (
    <Flex.Box y className={CSS.B("error-overlay")}>
      <Nav.Bar
        location="top"
        size="6.5rem"
        bordered
        data-tauri-drag-region
        background={2}
      >
        <Nav.Bar.Start>
          <OS.Controls
            visibleIfOS="macOS"
            forceOS={os}
            onClose={() => {
              if (Runtime.ENGINE === "tauri") void getCurrentWindow().close();
            }}
            onMinimize={() => {
              if (Runtime.ENGINE === "tauri") void getCurrentWindow().minimize();
            }}
            onMaximize={() => {
              if (Runtime.ENGINE === "tauri") void getCurrentWindow().maximize();
            }}
          />
          {os === "Windows" && (
            <Logo className="console-main-nav-top__logo" variant="icon" />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End justify="end">
          <OS.Controls
            visibleIfOS="Windows"
            forceOS={os}
            contrast={0}
            onClose={() => {
              if (Runtime.ENGINE === "tauri") void getCurrentWindow().close();
            }}
            onMinimize={() => {
              if (Runtime.ENGINE === "tauri") void getCurrentWindow().minimize();
            }}
            onMaximize={() => {
              if (Runtime.ENGINE === "tauri") void getCurrentWindow().maximize();
            }}
          />
        </Nav.Bar.End>
      </Nav.Bar>
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
