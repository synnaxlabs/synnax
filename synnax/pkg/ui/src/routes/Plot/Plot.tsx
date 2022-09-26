import {useEffect, useRef, useState} from "react";
import { Synnax, TimeSpan, TypedArray } from "@synnaxlabs/client";
import { Space, Header } from "@synnaxlabs/pluto";
import Sidebar from "../../lib/Sidebar/Sidebar";
import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";

const opts = {
    title: "Axis Control",
    width: 800,
    height: 600,
    scales: {
        x: {
            time: false,
            //	auto: false,
            //	range: [0, 6],
        },
        y: {
            auto: true,
        },
    },
    series: [
        {
            label: "x",
        },
        {
            label: "sin(x)",
            stroke: "#3774D0",
            strokeWith: 2,
        }
    ],
    axes: [
        {
            //	size: 30,
            label: "X Axis Label",
            labelSize: 20,
            stroke: "white",
            grid: {
                stroke: "rgba(255,255,255,0.1)",
                width: 1,
            }
        },
        {
            space: 50,
            side: 1,
            label: "Y Axis Label",
            labelGap: 8,
            labelSize: 8 + 12 + 8,
            stroke: "white",
            grid: {
                stroke: "rgba(255,255,255,0.1)",
                width: 1,
            }
        }
    ],
};

export default function Plot() {
  const [data, setData] = useState<TypedArray[]>([])
  const plotREf = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const client = new Synnax({ host: "localhost", port: 8080 });
    const f = async () => {
        const t0 = performance.now();
      const d = await client.data.read("1-1", 0, TimeSpan.Seconds(1000000));
      // @ts-ignore
        const t1 = performance.now();
        console.log("Call to doSomething took " + (t1 - t0) + " milliseconds.");
    console.log(d)
    let v = [
        Array.from({length: d.length}, (_, i) => i),
        Array.from(d),
    ];
    opts.height = plotREf.current.getBoundingClientRect().height - 100;
    opts.width = plotREf.current.getBoundingClientRect().width - 50;
    let uplot = new uPlot(opts, v, plotREf.current);
     return uplot.destroy
    };
    f();
  }, []);
  console.log(data)
  return (
       <Space empty direction="horizontal" >
          <Sidebar />
          <Space empty direction="vertical" style={{ flexGrow: 1}} align="stretch" >
              <Space empty direction="horizontal" style={{flexGrow: 1}}>
                  <Space empty direction="vertical" style={{flexGrow: 1}} size="large">
                      <div ref={plotREf} style={{height: "100%"}}></div>
                    </Space>
              </Space>
          </Space>
      </Space>
  )
}
