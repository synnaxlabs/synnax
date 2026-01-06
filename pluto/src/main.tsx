// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@/main.css";

import { type ReactElement } from "react";
import ReactDOM from "react-dom/client";

import { Pluto } from "@/pluto";
import { Showcase } from "@/showcase";

const Main = (): ReactElement => (
  <Pluto.Provider
    connParams={{
      host: "localhost",
      port: 9090,
      secure: false,
      username: "synnax",
      password: "seldon",
    }}
  >
    <Showcase />
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
