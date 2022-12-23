import { StrictMode } from "react";

import { Provider as DriftProvider } from "@synnaxlabs/drift";
import { Theming } from "@synnaxlabs/pluto";
import ReactDOM from "react-dom/client";

import { MainLayout } from "@/components";
import { ConnectCluster } from "@/features/cluster";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
} from "@/features/layout";
import { VisualizationLayoutRenderer } from "@/features/visualization";
import { DefineRange } from "@/features/workspace";
import { store } from "@/store";

import "@synnaxlabs/pluto/dist/style.css";
import "./index.css";

const layoutRenderers = {
  main: MainLayout,
  connectCluster: ConnectCluster,
  visualization: VisualizationLayoutRenderer,
  defineRange: DefineRange,
};

const MainUnderContext = (): JSX.Element => {
  const theme = useThemeProvider();
  return (
    <Theming.Provider {...theme}>
      <LayoutRendererProvider value={layoutRenderers}>
        <LayoutWindow />
      </LayoutRendererProvider>
    </Theming.Provider>
  );
};

const Main = (): JSX.Element => (
  <StrictMode>
    <DriftProvider store={store}>
      <MainUnderContext />
    </DriftProvider>
  </StrictMode>
);

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
