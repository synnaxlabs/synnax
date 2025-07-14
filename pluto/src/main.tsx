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

import { type ReactElement, useState } from "react";
import ReactDOM from "react-dom/client";

import { Align } from "@/align";
import { Button } from "@/button";
import { Pluto } from "@/pluto";
import { Ranger } from "@/ranger";

const RangeList = () => {
  const [selected, setSelected] = useState<string[]>([]);
  const [visible, setVisible] = useState(false);
  return (
    <Align.Space y style={{ padding: "10rem" }}>
      {visible ? (
        <Ranger.SelectMultiple value={selected} onChange={setSelected} />
      ) : null}
      <Button.Button onClick={() => setVisible(!visible)}>Toggle</Button.Button>
    </Align.Space>
  );
};

const Content = (): ReactElement => <RangeList />;

const Main = (): ReactElement => (
  <Pluto.Provider
    connParams={{
      host: "localhost",
      port: 9090,
      username: "synnax",
      password: "seldon",
    }}
  >
    <Content />
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
