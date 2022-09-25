import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";
import { ThemeProvider } from "@synnaxlabs/pluto";
import { synnaxDark, synnaxLight } from "@synnaxlabs/pluto";
import Login from "./routes/Login";
import Plot from "./routes/Plot/Plot";
import "@synnaxlabs/pluto/dist/style.css";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <ThemeProvider themes={[synnaxLight, synnaxDark]}>
    <Router>
      <Routes>
        <Route path="/" element={<Plot />} />
        <Route path="/login" element={<Login />} />
      </Routes>
    </Router>
   </ThemeProvider>
);
