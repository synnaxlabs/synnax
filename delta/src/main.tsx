// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StrictMode, useEffect, useState } from "react";

import { Provider } from "@synnaxlabs/drift";
import { Logo } from "@synnaxlabs/media";
import "@synnaxlabs/media/dist/style.css";
import {
  Theming,
  Triggers,
  Menu as PMenu,
  Space,
  Typography,
  useAsyncEffect,
  addOpacityToHex,
  Nav,
  Controls,
} from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import { appWindow, LogicalSize } from "@tauri-apps/api/window";
import ReactDOM from "react-dom/client";

import { useLoadTauriVersion } from "./features/version";
import { LayoutMain } from "./layouts/LayoutMain";
import { newStore } from "./store";

import { Menu } from "@/components";
import { ConnectCluster } from "@/features/cluster";
import { DocsLayoutRenderer } from "@/features/docs";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
  GetStarted,
  useErrorThemeProvider,
} from "@/features/layout";
import { VisLayoutRenderer } from "@/features/vis";
import { DefineRange } from "@/features/workspace";

import "./index.css";

const layoutRenderers = {
  main: LayoutMain,
  connectCluster: ConnectCluster,
  visualization: VisLayoutRenderer,
  defineRange: DefineRange,
  getStarted: GetStarted,
  docs: DocsLayoutRenderer,
};

export const DefaultContextMenu = (): JSX.Element => (
  <PMenu>
    <Menu.Item.HardReload />
  </PMenu>
);

const MainUnderContext = (): JSX.Element => {
  const theme = useThemeProvider();
  const menuProps = PMenu.useContextMenu();
  useLoadTauriVersion();
  return (
    <Theming.Provider {...theme}>
      <Triggers.Provider>
        <PMenu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
          <LayoutRendererProvider value={layoutRenderers}>
            <LayoutWindow />
          </LayoutRendererProvider>
        </PMenu.ContextMenu>
      </Triggers.Provider>
    </Theming.Provider>
  );
};

const Main = (): JSX.Element | null => {
  const store = newStore();
  return (
    <StrictMode>
      <Provider store={store} errorContent={() => <ErrorBoundary />}>
        <MainUnderContext />
      </Provider>
    </StrictMode>
  );
};

const ErrorBoundary = (): JSX.Element => {
  const theme = useErrorThemeProvider();
  useAsyncEffect(async () => {
    await appWindow.setResizable(false);
    await appWindow.setSize(new LogicalSize(800, 600));
    await appWindow.center();
    await appWindow.setFocus();
    await appWindow.setDecorations(true);
  });
  const handleClose = (): void => {
    void appWindow.close();
  };
  return (
    <Theming.Provider {...theme}>
      <Nav.Bar location="top" data-tauri-drag-region size="6rem">
        <Nav.Bar.Start className="delta-main-nav-top__start">
          <Controls
            visibleIfOS="MacOS"
            disabled={["minimize", "maximize"]}
            onClose={handleClose}
          />
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Controls visibleIfOS="Windows" />
        </Nav.Bar.End>
      </Nav.Bar>
      <Space.Centered
        size="large"
        style={{ height: "calc(100vh - 6rem - var(--os-border-offset, 0px))" }}
      >
        <Logo
          style={{
            width: 200,
            height: 200,
          }}
        />
        <div
          className="pluto--bordered"
          style={{
            borderColor: "var(--pluto-error-z)",
            padding: "2rem",
            borderRadius: "var(--pluto-border-radius)",
            backgroundColor: addOpacityToHex(theme.theme.colors.error.m1, 20),
          }}
        >
          <Typography.Text level="h4" style={{ width: 500 }}>
            It seems you have multiple Synnax windows open. Please close all other
            windows and try again.
          </Typography.Text>
        </div>
      </Space.Centered>
    </Theming.Provider>
  );
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
