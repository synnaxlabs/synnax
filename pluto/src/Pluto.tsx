import { PropsWithChildren, ReactElement } from "react";

import { Telem } from "./telem";

import { Aether } from "@/core/aether/main";
import { Worker } from "@/worker";

export interface PlutoProps extends PropsWithChildren {}

const WORKER_URL = new URL("./plutoWorker.ts", import.meta.url);

export const Pluto = ({ children }: PlutoProps): ReactElement => {
  return (
    <Worker.Provider url={WORKER_URL}>
      <Aether.Provider workerKey="vis">
        <Telem.Provider>{children}</Telem.Provider>
      </Aether.Provider>
    </Worker.Provider>
  );
};
