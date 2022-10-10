import React from "react";
import ReactDOM from "react-dom/client";
import { ThemeProvider, synnaxDark, synnaxLight } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import "./index.css";
import Main from "./Layouts/Main";

const data = {
  a: Array.from({ length: 10000 }, (_, i) => i),
  b: Array.from(
    { length: 10000 },
    (_, i) => Math.sin(i / 100) * 2 + Math.random() * 0.1
  ),
  c: Array.from(
    { length: 10000 },
    (_, i) => Math.cos(i / 190) + Math.random() * 0.1
  ),
  d: Array.from({ length: 10000 }, (_, i) => Math.sin(i / 200)),
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider themes={[synnaxLight, synnaxDark]}>
      <Main />
    </ThemeProvider>
  </React.StrictMode>
);
