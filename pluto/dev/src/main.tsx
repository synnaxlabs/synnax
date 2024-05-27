import ReactDOM from "react-dom/client";

import { Pluto, Canvas, LinePlot, telem, Range, TimeStamp } from "@synnaxlabs/pluto";

import { Channel, TimeSpan } from "@synnaxlabs/pluto";

import WorkerURL from "./worker?worker&url";

import "@synnaxlabs/pluto/dist/style.css";

const Plot = () => {
  const start = TimeStamp.now().valueOf();
  const xData = telem.fixedArray({
    data: [
      new Float32Array(
        Array.from({ length: 1000 }, (_, i) => Number(TimeStamp.seconds(i).valueOf())),
      ),
    ],
    offsets: [Number(start.valueOf())],
  });
  const yData = telem.fixedArray({
    data: [new Float32Array(Array.from({ length: 1000 }, (_, i) => Math.sin(i / 30)))],
  });
  return (
    <LinePlot.LinePlot>
      <LinePlot.XAxis bounds={{ lower: 0, upper: 10 }} type="time">
        <Range.Provider />
        <LinePlot.YAxis bounds={{ lower: 0, upper: 10 }}>
          <LinePlot.Line
            label="Line 1"
            color="#b57edc"
            x={xData}
            y={yData}
            strokeWidth={2}
          />
        </LinePlot.YAxis>
      </LinePlot.XAxis>
      <LinePlot.Viewport>
        <LinePlot.Tooltip.Tooltip />
      </LinePlot.Viewport>
    </LinePlot.LinePlot>
  );
};

const Main = () => (
  <Pluto.Provider
    workerURL={WorkerURL}
    theming={{ theme: { colors: { primary: "#b57edc" } } }}
    connParams={{
      host: "localhost",
      port: 9090,
      username: "synnax",
      password: "seldon",
      secure: false,
    }}
  >
    <Canvas.Canvas
      style={{
        position: "fixed",
        top: 0,
        left: 0,
        width: "100%",
        height: "100%",
        background: "var(--pluto-gray-l0)",
      }}
    >
      <Plot />
    </Canvas.Canvas>
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
