import { PlottingContextProvider, Space, Plot } from "@synnaxlabs/pluto";
import Sidebar from "../../lib/Sidebar/Sidebar";
import { useEffect } from "react";

export default function PlotMatrix() {
  console.log("HELLO");
  useEffect(() => {
    const numPoints = 16;
    // Allocate numPoints
    const alloc = performance.now();
    const buffer = new ArrayBuffer(numPoints);
    const allocTime = performance.now() - alloc;
    console.log(`Allocated ${numPoints} in ${allocTime}ms`);
    const next = performance.now();
    const nextData = new Float32Array(buffer);
    nextData[0] = 1;
    const nextTime = performance.now() - next;
    console.log(`Converted ${numPoints} in ${nextTime}ms`);
    console.log(buffer);
  }, []);
  return (
    <Space empty direction="vertical" style={{ flexGrow: 1 }}>
      <Plot />
    </Space>
  );
}
