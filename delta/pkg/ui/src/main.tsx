import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";
import { ThemeProvider } from "@arya-analytics/pluto";
import { aryaDark, aryaLight } from "@arya-analytics/pluto";
import Login from "./routes/Login";
import "@arya-analytics/pluto/dist/style.css";
import Cluster from "./routes/Cluster";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider themes={[aryaLight, aryaDark]}>
      <Router>
        <Routes>
          <Route path="/" element={<Cluster />} />
          <Route path="/login" element={<Login />} />
        </Routes>
      </Router>
    </ThemeProvider>
  </React.StrictMode>
);
