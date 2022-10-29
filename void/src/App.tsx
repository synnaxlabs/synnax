import "./index.css";
import "@synnaxlabs/pluto/dist/style.css";
import { Theming } from "@synnaxlabs/pluto";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
} from "@/features/layout";
import { MainLayout } from "./components";
import { ConnectCluster } from "@/features/cluster";
import { Plot } from "@/features/visualization";

const layoutRenderers = {
  main: MainLayout,
  connectCluster: ConnectCluster,
  plot: Plot,
};

export const App = () => {
  const theme = useThemeProvider();
  return (
    <LayoutRendererProvider value={layoutRenderers}>
      <Theming.Provider {...theme}>
        <LayoutWindow />
      </Theming.Provider>
    </LayoutRendererProvider>
  );
};
