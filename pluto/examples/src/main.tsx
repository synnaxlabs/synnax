import ReactDOM from "react-dom/client";
import App from "./App.tsx";

import WorkerURL from "./worker?worker&url";

import { Pluto, Canvas } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";

const Main = () => {
  return (
    <Pluto.Provider
      workerURL={WorkerURL}
      connParams={{
        host: "localhost",
        port: 9090,
        username: "synnax",
        password: "seldon",
      }}
    >
      <Canvas.Canvas
        style={{
          position: "fixed",
          top: 0,
          left: 0,
          width: "100%",
          height: "100%",
        }}
      >
        <App />
      </Canvas.Canvas>
    </Pluto.Provider>
  );
};

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
