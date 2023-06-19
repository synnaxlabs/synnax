// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt

import { RoutedWorker } from "@synnaxlabs/x";

import { AetherComponentConstructor, render } from "@/core/aether/worker";
import { LineGL } from "@/core/vis/Line/LineGL";
import { LinePlot, XAxis, YAxis } from "@/core/vis/LinePlot/worker";
import { Value } from "@/core/vis/pid/Value/worker";
import { Canvas } from "@/core/vis/WorkerCanvas";
import { Telem } from "@/telem/worker";

const w = new RoutedWorker((data, transfer) => postMessage(data, "/", transfer));
onmessage = (e) => w.handle(e);

const REGISTRY: Record<string, AetherComponentConstructor> = {
  [Telem.TYPE]: (u) => new Telem(u),
  [LinePlot.TYPE]: (u) => new LinePlot(u),
  [XAxis.TYPE]: (u) => new XAxis(u),
  [YAxis.TYPE]: (u) => new YAxis(u),
  [LineGL.TYPE]: (u) => new LineGL(u),
  [Canvas.TYPE]: (u) => new Canvas(u),
  [Value.TYPE]: (u) => new Value(u),
};

render(w.route("vis"), REGISTRY);
