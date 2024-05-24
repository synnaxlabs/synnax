import { Layout } from "@/layout";
import { Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { ReactElement, useEffect, useState } from "react";
import "@/range/MetaData.css";

export const metaDataWindowLayout: Layout.State = {
  key: "metaData",
  windowKey: "metaData",
  type: "metaData",
  name: "Meta Data",
  location: "window",
  window: {
    resizable: false,
    size: { height: 430, width: 650 },
    navTop: true,
  },
};

export const MetaData: Layout.Renderer = ({ layoutKey }): ReactElement => {
  const client = Synnax.use();
  const [data, setData] = useState<Record<string, string>>({});
  useEffect(() => {
    const i = setInterval(async () => {
      if (client == null) return;
      const rng = await client?.ranges.retrieve(layoutKey);
      setData(await rng.kv.list());
    }, 200);
    return () => clearInterval(i);
  }, [client]);
  return (
    <div>
      <table>
        <thead>
          <tr>
            <th>Key</th>
            <th>Value</th>
          </tr>
        </thead>
        <tbody>
          {Object.entries(data).map(([key, value], index) => (
            <tr key={index}>
              <td>{key}</td>
              <td>{value}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
