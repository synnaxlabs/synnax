// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StrictMode } from "react";

import { Provider } from "@synnaxlabs/drift";
import { Logo } from "@synnaxlabs/media";
import {
  Theming,
  Triggers,
  Menu as PMenu,
  Space,
  Typography,
  Nav,
  Controls,
  Haul,
} from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";
import ReactDOM from "react-dom/client";

import "./index.css";

import { ConnectCluster } from "@/cluster";
import { Menu } from "@/components";
import { DocsLayoutRenderer } from "@/docs";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
  GetStarted,
  useErrorThemeProvider,
} from "@/layout";
import { LayoutMain } from "@/layouts/LayoutMain";
import { store } from "@/store";
import { useLoadTauriVersion } from "@/version";
import { VisLayoutRenderer } from "@/vis";
import { DefineRange } from "@/workspace";

import "@synnaxlabs/media/dist/style.css";
import "@synnaxlabs/pluto/dist/style.css";

const layoutRenderers = {
  main: LayoutMain,
  connectCluster: ConnectCluster,
  visualization: VisLayoutRenderer,
  defineRange: DefineRange,
  getStarted: GetStarted,
  docs: DocsLayoutRenderer,
};

export const DefaultContextMenu = (): ReactElement => (
  <PMenu>
    <Menu.Item.HardReload />
  </PMenu>
);

const MainUnderContext = (): ReactElement => {
  const theme = useThemeProvider();
  const menuProps = PMenu.useContextMenu();
  useLoadTauriVersion();
  return (
    <Theming.Provider {...theme}>
      <Haul.Provider>
        <Triggers.Provider>
          <PMenu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
            <LayoutRendererProvider value={layoutRenderers}>
              <LayoutWindow />
            </LayoutRendererProvider>
          </PMenu.ContextMenu>
        </Triggers.Provider>
      </Haul.Provider>
    </Theming.Provider>
  );
};

const Main = (): ReactElement | null => {
  return (
    <StrictMode>
      <Provider store={store} errorContent={(e) => <ErrorBoundary err={e} />}>
        <MainUnderContext />
      </Provider>
    </StrictMode>
  );
};

const ErrorBoundary = ({ err }: { err: Error }): ReactElement => {
  const theme = useErrorThemeProvider();
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
          <Typography.Text level="h4" style={{ width: 500, textAlign: "center" }}>
            {JSON.stringify(err.message)}
          </Typography.Text>
        </div>
      </Space.Centered>
    </Theming.Provider>
  );
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
