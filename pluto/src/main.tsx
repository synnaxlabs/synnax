// Copyright 2025 Synnax Labs, Inc.
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

import { Align } from "@/align";
import { CSS } from "@/css";
import { Pluto } from "@/pluto";

const Main = (): ReactElement => (
  <Pluto.Provider>
    <div style={{ padding: "2rem" }}>
      <Align.Space x className={CSS.B("stack")} />
    </div>
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
