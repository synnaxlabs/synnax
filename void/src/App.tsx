import { StrictMode } from "react";
import { Provider as DriftProvider } from "@synnaxlabs/drift";
import store from "./store";
import "./index.css";
import "@synnaxlabs/pluto/dist/style.css";
import { Theme } from "@synnaxlabs/pluto";
import { LayoutRendererProvider, LayoutWindow } from "@/features/layout";
import { MainLayout } from "./components/MainLayout";
import { ConnectCluster } from "@/features/cluster";

const layoutRenderers = {
  main: MainLayout,
  connectCluster: ConnectCluster,
};

export const App = () => {
  return (
    <StrictMode>
      <DriftProvider store={store}>
        <Theme.Provider
          themes={[Theme.themes.synnaxDark, Theme.themes.synnaxLight]}
        >
          <LayoutRendererProvider value={layoutRenderers}>
            <LayoutWindow />
          </LayoutRendererProvider>
        </Theme.Provider>
      </DriftProvider>
    </StrictMode>
  );
};
