import { StrictMode } from "react";
import { Provider as DriftProvider } from "@synnaxlabs/drift";
import store from "./store";
import "./index.css";
import "@synnaxlabs/pluto/dist/style.css";
import { Theme } from "@synnaxlabs/pluto";
import { LayoutRenderersProvider } from "@/features/layout";

export const App = () => {
  return (
    <StrictMode>
      <DriftProvider store={store}>
        <Theme.Provider
          themes={[Theme.themes.synnaxDark, Theme.themes.synnaxLight]}
        >
          <LayoutRenderersProvider value={{}}></LayoutRenderersProvider>
        </Theme.Provider>
      </DriftProvider>
    </StrictMode>
  );
};
