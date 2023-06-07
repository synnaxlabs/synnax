import { RoutedWorker } from "@synnaxlabs/x";

import { AetherRoot } from "@/core/aether/worker";
import { newBootstrapFn } from "@/core/vis/WorkerCanvas";
import { TelemWorker } from "@/telem/worker";

const w = new RoutedWorker((data, transfer = []) => postMessage(data, "/", transfer));
onmessage = (e) => w.handle(e);
const telem = new TelemWorker(w.route("telem"));
const fn = newBootstrapFn(telem);
// eslint-disable-next-line no-new
new AetherRoot(w.route("vis"), fn);
