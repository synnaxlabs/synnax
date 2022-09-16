import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";
import { ThemeProvider, synnaxDark, synnaxLight } from "@synnaxlabs/pluto";
import Login from "./routes/Login";
import "@synnaxlabs/pluto/dist/style.css";
import Cluster from "./routes/Cluster";
import PlotMatrix from "./routes/Plot";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <ThemeProvider themes={[synnaxDark]}>
    <Router>
      <Routes>
        <Route path="/" element={<PlotMatrix />} />
        <Route path="/login" element={<Login />} />
      </Routes>
    </Router>
  </ThemeProvider>
);
