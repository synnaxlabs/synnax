// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/error/Overlay.css";

import { Logo } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  CSS as PCSS,
  Nav,
  OS,
  Status,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { type PropsWithChildren, type ReactElement, useEffect } from "react";
import {
  ErrorBoundary,
  type ErrorBoundaryProps,
  type FallbackProps,
} from "react-error-boundary";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { NAV_SIZES } from "@/layouts/constants";
import { Persist } from "@/persist";
import { CLEAR_STATE, REVERT_STATE } from "@/persist/state";

export interface OverlayProps extends PropsWithChildren {}

const messageTranslation: Record<string, string> = {
  "[persist] - windows open":
    "It seems like you have Synnax open from multiple windows. Please close all other windows and reopen Synnax.",
};

const FallbackRenderWithStore: ErrorBoundaryProps["fallbackRender"] = ({ error }) => {
  const dispatch = useDispatch();
  const handleTryAgain = (): void => {
    dispatch(REVERT_STATE);
  };

  const handleClear = (): void => {
    dispatch(CLEAR_STATE);
  };

  return (
    <FallBackRenderContent
      onClear={handleClear}
      onTryAgain={handleTryAgain}
      error={error}
    />
  );
};
const FallbackRenderWithoutStore: ErrorBoundaryProps["fallbackRender"] = ({
  error,
}) => <FallBackRenderContent onClear={Persist.hardClearAndReload} error={error} />;

type FallbackRenderContentProps = Pick<FallbackProps, "error"> & {
  onTryAgain?: () => void;
  onClear?: () => void;
};

const FallBackRenderContent = ({
  onTryAgain,
  onClear,
  error,
}: FallbackRenderContentProps): ReactElement => {
  const os = OS.use();
  useEffect(() => {
    // grab the prefers-color-scheme media query
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
    void getCurrentWindow().show();
  }, []);
  return (
    <Align.Space direction="y" className={CSS.B("error-overlay")}>
      <Nav.Bar location="top" size={NAV_SIZES.top} className="console-main-nav-top">
        <Nav.Bar.Start className="console-main-nav-top__start">
          <OS.Controls
            className="console-controls--macos"
            visibleIfOS="macOS"
            onClose={() => {
              void getCurrentWindow().close();
            }}
            onMinimize={() => {
              void getCurrentWindow().minimize();
            }}
            onMaximize={() => {
              void getCurrentWindow().maximize();
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
            onClose={() => {
              void getCurrentWindow().close();
            }}
            onMinimize={() => {
              void getCurrentWindow().minimize();
            }}
            onMaximize={() => {
              void getCurrentWindow().maximize();
            }}
          />
        </Nav.Bar.End>
      </Nav.Bar>

      <Align.Center role="alert">
        <Align.Space direction="x" className={CSS.B("dialog")} size={20}>
          <Logo variant="icon" />
          <Align.Space direction="y" align="start" className={CSS.B("details")}>
            <Text.Text level="h1">Something went wrong</Text.Text>
            <Status.Text variant="error" hideIcon level="h3">
              {messageTranslation[error.message] ?? error.message}
            </Status.Text>
            <Text.Text className={CSS.B("stack")} level="p">
              {error.stack}
            </Text.Text>
            <Align.Space direction="x">
              {onTryAgain && (
                <Button.Button onClick={onTryAgain}>Try again</Button.Button>
              )}
              {onClear && (
                <Button.Button onClick={onClear} variant="outlined">
                  Clear Storage and Hard Reset
                </Button.Button>
              )}
            </Align.Space>
          </Align.Space>
        </Align.Space>
      </Align.Center>
    </Align.Space>
  );
};

const fallbackRenderWithStore = componentRenderProp(FallbackRenderWithStore);
const fallbackRenderWithoutStore = componentRenderProp(FallbackRenderWithoutStore);

export const OverlayWithStore = ({ children }: OverlayProps): ReactElement => (
  <ErrorBoundary fallbackRender={fallbackRenderWithStore}>{children}</ErrorBoundary>
);

export const OverlayWithoutStore = ({ children }: OverlayProps): ReactElement => (
  <ErrorBoundary fallbackRender={fallbackRenderWithoutStore}>{children}</ErrorBoundary>
);
