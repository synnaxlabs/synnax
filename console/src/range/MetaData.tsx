// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/MetaData.css";

import { Synnax } from "@synnaxlabs/pluto";
import { ReactElement, useEffect, useState } from "react";

import { Layout } from "@/layout";

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
