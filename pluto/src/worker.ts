// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt

import { RoutedWorker } from "@synnaxlabs/x";

import { AetherComponentRegistry, render } from "@/core/aether/worker";
import { LineGL } from "@/core/vis/Line/LineGL";
import { LinePlot, XAxis, YAxis } from "@/core/vis/LinePlot/worker";
import { PID } from "@/core/vis/pid/worker";
import { Value } from "@/core/vis/Value/worker";
import { Valve } from "@/core/vis/Valve/worker";
import { Canvas } from "@/core/vis/WorkerCanvas";
import { Telem } from "@/telem/worker";

const w = new RoutedWorker((data, transfer) => postMessage(data, "/", transfer));
onmessage = (e) => w.handle(e);

const REGISTRY: AetherComponentRegistry = {
  [Telem.TYPE]: (u) => new Telem(u),
  [LinePlot.TYPE]: (u) => new LinePlot(u),
  [XAxis.TYPE]: (u) => new XAxis(u),
  [YAxis.TYPE]: (u) => new YAxis(u),
  [LineGL.TYPE]: (u) => new LineGL(u),
  [Canvas.TYPE]: (u) => new Canvas(u),
  [Value.TYPE]: (u) => new Value(u),
  [Valve.TYPE]: (u) => new Valve(u),
  [PID.TYPE]: (u) => new PID(u),
};

render(w.route("vis"), REGISTRY);
