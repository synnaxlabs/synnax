import React from "react";
import ReactDOM from "react-dom/client";
import { ThemeProvider, synnaxDark, synnaxLight } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import "./index.css";
import Main from "./Layouts/Main";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider themes={[synnaxLight, synnaxDark]}>
      <Main />
    </ThemeProvider>
  </React.StrictMode>
);
