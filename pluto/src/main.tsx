// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { box } from "@synnaxlabs/x";
import { useState } from "react";
import { createRoot } from "react-dom/client";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { Color } from "@/color";
import { useAsyncEffect } from "@/hooks";
import { Pluto } from "@/pluto";
import { Synnax } from "@/synnax";
import { Table } from "@/table";
import { telem } from "@/telem/aether";
import { Text } from "@/text";
import { Canvas } from "@/vis/canvas";
import { Value as CoreValue } from "@/vis/value";
import WorkerURL from "@/workMain?worker&url";

const Value = Aether.wrap("Value", ({ aetherKey }) => {
  const [b, setB] = useState<box.Box>(box.ZERO);
  const [cKey, setCKey] = useState<channel.Key>(0);

  const client = Synnax.use();

  useAsyncEffect(async () => {
    if (client == null) return;
    const ch = await client.channels.retrieve("stream_write_example_data_1");
    setCKey(ch.key);
  }, [client?.key]);

  const props = CoreValue.use({
    aetherKey,
    telem: telem.sourcePipeline("string", {
      segments: {
        valueStream: telem.streamChannelValue({ channel: cKey }),
        stringifier: telem.stringifyNumber({ precision: 2 }),
      },
      connections: [{ from: "valueStream", to: "stringifier" }],
      outlet: "stringifier",
    }),
    box: b,
    level: "p",
  });

  const ref = Canvas.useRegion((b) => setB(b));
  return <div ref={ref} style={{ width: "100%", height: "5rem" }}></div>;
});

const Main = () => (
  <Pluto.Provider
    connParams={{
      host: "demo.synnaxlabs.com",
      port: 9090,
      username: "synnax",
      password: "seldon",
      secure: true,
    }}
    workerURL={WorkerURL}
    theming={{ defaultTheme: "synnaxDark" }}
  >
    <Color.G />
  </Pluto.Provider>
);

createRoot(document.getElementById("root")!).render(<Main />);
