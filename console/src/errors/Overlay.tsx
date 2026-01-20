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
import { Button, CSS as PCSS, Errors, Flex, Nav, OS, Theming } from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
} from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Persist } from "@/persist";
import { CLEAR_STATE, REVERT_STATE } from "@/persist/state";
import { Runtime } from "@/runtime";

export interface OverlayProps extends PropsWithChildren {}

const FallbackRenderWithStore = ({ error }: Errors.FallbackProps): ReactElement => {
  const dispatch = useDispatch();
  const handleTryAgain = useCallback((): void => {
    dispatch(REVERT_STATE);
  }, [dispatch]);
  const handleClear = useCallback((): void => {
    dispatch(CLEAR_STATE);
  }, [dispatch]);
  return (
    <FallBackRenderContent
      onClear={handleClear}
      onTryAgain={handleTryAgain}
      error={error}
    />
  );
};

const FallbackRenderWithoutStore = ({ error }: Errors.FallbackProps): ReactElement => (
  <FallBackRenderContent onClear={Persist.hardClearAndReload} error={error} />
);

interface FallbackRenderContentProps {
  error: Error;
  onTryAgain?: () => void;
  onClear: () => void;
}

const FallBackRenderContent = ({
  onTryAgain,
  onClear,
  error,
}: FallbackRenderContentProps): ReactElement => {
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
        className="console-main-nav-top"
        bordered
        data-tauri-drag-region
      >
        <Nav.Bar.Start className="console-main-nav-top__start">
          <OS.Controls
            className="console-controls--macos"
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
        <Nav.Bar.End className="console-main-nav-top__end" justify="end">
          <OS.Controls
            className="console-controls--windows"
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
        variant="full"
        showLogo
      >
        <Flex.Box x>
          {onTryAgain != null && (
            <Button.Button variant="filled" onClick={onTryAgain}>
              Reload Console
            </Button.Button>
          )}
          <Button.Button onClick={onClear}>
            Clear storage and reload Console
          </Button.Button>
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
