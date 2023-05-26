import { PropsWithChildren, useState } from "react";

import { PIDContextC, PIDEngine, RenderSet, workerRenderer } from "./PIDContext";

export interface PIDContextProviderProps extends PropsWithChildren<{}> {
  engine: PIDEngine;
  renderers: Record<string, RenderSet<unknown>>;
}

export const PIDProvider = ({ children }: PIDContextProviderProps): ReactElement => {
  const [worker, setWorker] = useState<Worker | null>(null);
  const [canbasTransferred, setCanvasTransferred] = useState<boolean>(false);
  return (
    <PIDContextC.Provider value={{ render: workerRenderer(worker) }}>
      <canvas
        style={{ position: "fixed" }}
        height={window.innerHeight}
        width={window.innerWidth}
        ref={(el: HTMLCanvasElement) => {
          if (el == null || canbasTransferred) return;
          setCanvasTransferred(true);
          const worker = new Worker(new URL("./worker.ts", import.meta.url), {
            type: "module",
          });
          el.width = window.innerWidth * window.devicePixelRatio;
          el.height = window.innerHeight * window.devicePixelRatio;
          const offscreen = el.transferControlToOffscreen();
          worker.postMessage(
            {
              type: "init",
              props: { canvas: offscreen, scale: window.devicePixelRatio },
            },
            [offscreen]
          );
          setWorker(worker);
        }}
      ></canvas>
      {children}
    </PIDContextC.Provider>
  );
};
