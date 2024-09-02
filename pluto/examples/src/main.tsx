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

import { Pluto } from "@synnaxlabs/pluto";
import ReactDOM from "react-dom/client";

import { ChildRanges } from "./ChildRanges";
import { ExampleTimeline } from "./TImeline";
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
    <ExampleTimeline />
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
