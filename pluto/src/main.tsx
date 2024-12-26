// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";

import { TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import ReactDOM from "react-dom/client";

import { Input } from "@/input";
import { Pluto } from "@/pluto";

const Main = (): ReactElement => {
  const [value, setValue] = useState<number>(Number(TimeStamp.now().valueOf()));
  return (
    <div style={{ padding: "2rem" }}>
      <Pluto.Provider>
        <Input.DateTime value={value} onChange={setValue} />
      </Pluto.Provider>
    </div>
  );
};

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
