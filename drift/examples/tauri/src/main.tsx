// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "./index.css";

import { Provider } from "@synnaxlabs/drift";
import React from "react";
import ReactDOM from "react-dom";

import App from "./App";
import promise from "./store";

const Main = (): ReactElement => {
  return (
    <React.StrictMode>
      <Provider store={promise}>
        <App />
      </Provider>
    </React.StrictMode>
  );
};

ReactDOM.render(<Main />, document.getElementById("root"));
