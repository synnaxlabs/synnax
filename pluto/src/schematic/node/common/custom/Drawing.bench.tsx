// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { flushSync } from "react-dom";
import { createRoot, type Root } from "react-dom/client";
import { test } from "vitest";

import { Custom } from "@/schematic/node/common/custom";
import { Primitive } from "@/schematic/node/common/primitive";

// Times a schematic of 300 custom symbols mounting, through the parse, the sanitizer,
// and the first draw of their paints.

const COUNT = 300;

const SPEC: schematic.symbol.Spec = {
  drawing: {
    type: "svg",
    markup:
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 87 42">' +
      '<path class="body" fill="none" stroke="#cfd1d4" stroke-width="2" ' +
      'd="M43.5 21L6.4 2.2C4.4 1.2 2 2.6 2 4.9V37.1C2 39.4 4.4 40.8 6.4 39.8L43.5 21Z' +
      'M43.5 21L80.6 2.2C82.6 1.2 85 2.6 85 4.9V37.1C85 39.4 82.6 40.8 80.6 39.8L43.5 21Z"/>' +
      "</svg>",
  },
  parts: [],
  handles: [],
  states: [],
  commands: [],
  looks: [],
  interactions: [],
  variant: "static",
  scale: 1,
  strokeScaled: false,
  previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
};

const PAINTS: Custom.Paint[] = [{ selectors: [".body"], stroke: [207, 209, 212, 1] }];

const SYMBOLS = Array.from({ length: COUNT }, (_, i) => (
  <Primitive.Div key={i}>
    <Custom.Drawing spec={SPEC} paints={PAINTS} />
  </Primitive.Div>
));

test(`mount ${COUNT} custom symbols`, async ({ bench }) => {
  let host: HTMLElement;
  let root: Root;
  await bench(
    `should mount ${COUNT} symbols`,
    {
      beforeEach: () => {
        host = document.createElement("div");
        document.body.append(host);
        root = createRoot(host);
      },
      afterEach: () => {
        root.unmount();
        host.remove();
      },
    },
    () => flushSync(() => root.render(SYMBOLS)),
  ).run({ time: 2000 });
});
