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

import { TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import ReactDOM from "react-dom/client";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Input } from "@/input";
import { Pluto } from "@/pluto";
import { Tag } from "@/tag";

interface TagSpec {
  color: string;
  children: string;
}

const TAGS: TagSpec[] = [
  { color: "#00FF00", children: "Hotfire" },
  { color: "#0000FF", children: "Scrubbed" },
  { color: "#0000FF", children: "TS1" },
  { color: "#0000FF", children: "Qual" },
];

const Main = (): ReactElement => {
  const [value, setValue] = useState<number>(Number(TimeStamp.now().valueOf()));
  return (
    <Pluto.Provider>
      <div style={{ padding: "2rem" }}>
        <Align.Space x className={CSS.B("stack")}></Align.Space>
      </div>
    </Pluto.Provider>
  );
};

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
