import { RoutedWorker } from "@synnaxlabs/x";

import { TelemWorker } from "@/telem/worker";

const w = new RoutedWorker(
  (data, transfer = []) => postMessage(data, "/", transfer),
  (handler: (data: any) => void) => (onmessage = handler)
);

const telem = new TelemWorker(w.route("telem"));
