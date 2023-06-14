// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RoutedWorker } from "@synnaxlabs/x";

import { AetherRoot } from "@/core/aether/worker";
import { newBootstrapFn } from "@/core/vis/WorkerCanvas";
import { TelemService } from "@/telem/worker";

const w = new RoutedWorker((data, transfer = []) => postMessage(data, "/", transfer));
onmessage = (e) => w.handle(e);

const telem = new TelemService(w.route("telem"));
const fn = newBootstrapFn(telem);

AetherRoot.render(w.route("vis"), fn);
