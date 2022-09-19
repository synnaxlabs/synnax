import React, { ReactElement } from "react";
import ReactDOM from "react-dom/client";
import TechnicalIntroduction from "./presentations/1-220917-technical-introduction";
import { BrowserRouter as Router, Route, Routes, Link } from "react-router-dom";
import { ThemeProvider, synnaxLight, synnaxDark } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";

const presentations: {
  title: string;
  path: string;
  component: ReactElement;
}[] = [
  {
    title: "Technical Introduction",
    path: "/1-220917-technical-introduction",
    component: <TechnicalIntroduction />,
  },
];

const Home = () => {
  return (
    <ul>
      {presentations.map((presentation) => (
        <li>
          <Link key={presentation.path} to={presentation.path}>
            {presentation.title}
          </Link>
        </li>
      ))}
    </ul>
  );
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <div className="presentations">
    <ThemeProvider themes={[synnaxLight]}>
      <Router>
        <Routes>
          {presentations.map((presentation) => (
            <Route
              key={presentation.path}
              path={presentation.path}
              element={presentation.component}
            />
          ))}
          <Route path="/" element={<Home />} />
        </Routes>
      </Router>
    </ThemeProvider>
  </div>
);
