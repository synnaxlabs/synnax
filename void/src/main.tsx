import React from "react";
import ReactDOM from "react-dom/client";
import { Theme } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import "./index.css";
import Main from "./Layouts/Main";
import { Provider } from "@synnaxlabs/drift";
import store from "./store";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import ConnectCluster from "./cluster/ConnectCluster";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <Router>
      <Provider store={store}>
        <Theme.Provider
          themes={[Theme.themes.synnaxDark, Theme.themes.synnaxLight]}
        >
          <Routes>
            <Route path="/" element={<Main />} />
            <Route path="/cluster/connect" element={<ConnectCluster />} />
          </Routes>
        </Theme.Provider>
      </Provider>
    </Router>
  </React.StrictMode>
);
