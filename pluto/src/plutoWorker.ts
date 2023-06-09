import { Synnax } from "@synnaxlabs/client";
import { RoutedWorker } from "@synnaxlabs/x";

import { AetherRoot } from "@/core/aether/worker";
import { newBootstrapFn } from "@/core/vis/WorkerCanvas";
import { Client } from "@/telem/client";
import { RangeTelemFactory } from "@/telem/range/worker";
import { StaticTelemFactory } from "@/telem/static/worker";
import { TelemService } from "@/telem/worker";

const w = new RoutedWorker((data, transfer = []) => postMessage(data, "/", transfer));
onmessage = (e) => w.handle(e);
const staticFactory = new StaticTelemFactory();
const sy = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
});
const rangeFactory = new RangeTelemFactory(new Client(sy));
const factories = [staticFactory, rangeFactory];
const telem = new TelemService(w.route("telem"), factories);
const fn = newBootstrapFn(telem);
// eslint-disable-next-line no-new
new AetherRoot(w.route("vis"), fn);
