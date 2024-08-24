// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@synnaxlabs/pluto/dist/style.css";
import "./main.css";

import { Align, Canvas, Pluto, Tag } from "@synnaxlabs/pluto";
import { Channel, TimeSpan } from "@synnaxlabs/pluto";
import ReactDOM from "react-dom/client";

import WorkerURL from "./worker?worker&url";

const Main = () => (
  <Pluto.Provider
    workerURL={WorkerURL}
    connParams={{
      host: "demo.synnaxlabs.com",
      port: 9090,
      username: "synnax",
      password: "seldon",
      secure: true,
    }}
  >
    <Align.Center
      direction="x"
      style={{
        width: "100vw",
        background:
          "linear-gradient(45deg, var(--pluto-primary-z), var(--pluto-error-z))",
      }}
    >
      <Align.Space
        direction="x"
        style={{
          background: "var(--pluto-gray-l1)",
          padding: "2rem",
          border: "var(--pluto-border-l4)",
          borderRadius: "1rem",
        }}
      >
        <Tag.Tag color="var(--pluto-secondary-z)">Success</Tag.Tag>
        <Tag.Tag color="#e70ceb">Completed</Tag.Tag>
        <Tag.Tag color="var(--pluto-primary-z)">Qualification</Tag.Tag>
        <Tag.Tag color="#eb950c">V1</Tag.Tag>
      </Align.Space>
    </Align.Center>
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
