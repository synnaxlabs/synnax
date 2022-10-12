import ReactDOM from "react-dom/client";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";
import { Theme } from "@synnaxlabs/pluto";
import Login from "./routes/Login";
import "@synnaxlabs/pluto/dist/style.css";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <Theme.Provider themes={[Theme.themes.synnaxLight, Theme.themes.synnaxDark]}>
    <Router>
      <Routes>
        <Route path="/login" element={<Login />} />
      </Routes>
    </Router>
  </Theme.Provider>
);
