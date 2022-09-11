import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";
import { ThemeProvider, aryaDark, aryaLight } from "@arya-analytics/pluto";
import Login from "./routes/Login";
import "@arya-analytics/pluto/dist/style.css";
import PlotMatrix from "./routes/Plot";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <ThemeProvider themes={[aryaDark]}>
    <Router>
      <Routes>
        <Route path="/" element={<PlotMatrix />} />
        <Route path="/login" element={<Login />} />
      </Routes>
    </Router>
  </ThemeProvider>
);
