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
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Pluto } from "@/pluto";
import { Text } from "@/text";

const Main = (): ReactElement => {
  const [value, setValue] = useState("");
  return (
    <Pluto.Provider>
      <Align.Center x>
        <Input.Text
          value={value}
          onChange={setValue}
          placeholder={
            <>
              <Icon.Search />
              Search
            </>
          }
        >
          <Button.Button variant="outlined">
            <Icon.Search />
          </Button.Button>
        </Input.Text>
      </Align.Center>
    </Pluto.Provider>
  );
};

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
