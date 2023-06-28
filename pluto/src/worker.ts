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
import { AetherCanvas } from "@/core/vis/Canvas/aether";
import { LineGL } from "@/core/vis/Line/LineGL";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { AetherPID } from "@/core/vis/pid/aether";
import { AetherValue } from "@/core/vis/Value/aether";
import { Valve } from "@/core/vis/Valve/aether";
import { Telem } from "@/telem/TelemProvider/aether";

// @ts-expect-error
const w = new RoutedWorker((data, transfer) => postMessage(data, transfer));
onmessage = (e) => w.handle(e);

const REGISTRY: AetherComponentRegistry = {
  ...AetherLinePlot.REGISTRY,
  ...AetherCanvas.REGISTRY,
  [Telem.TYPE]: (u) => new Telem(u),
  [LineGL.TYPE]: (u) => new LineGL(u),
  [AetherValue.TYPE]: (u) => new AetherValue(u),
  [Valve.TYPE]: (u) => new Valve(u),
  [AetherPID.TYPE]: (u) => new AetherPID(u),
};

render(w.route("vis"), REGISTRY);
