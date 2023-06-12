import { RoutedWorker } from "@synnaxlabs/x";

import { AetherRoot } from "@/core/aether/worker";
import { newBootstrapFn } from "@/core/vis/WorkerCanvas";
import { TelemService } from "@/telem/worker";

const w = new RoutedWorker((data, transfer = []) => postMessage(data, "/", transfer));
onmessage = (e) => w.handle(e);

const telem = new TelemService(w.route("telem"));
const fn = newBootstrapFn(telem);

AetherRoot.render(w.route("vis"), fn);
