import { PropsWithChildren, ReactElement } from "react";

import { Telem } from "./telem";

import { Worker } from "@/worker";

export interface PlutoProps extends PropsWithChildren {}

const WORKER_URL = new URL("./plutoWorker.ts", import.meta.url);

export const Pluto = ({ children }: PlutoProps): ReactElement => {
  return (
    <Worker.Provider url={WORKER_URL}>
      <Telem.Provider></Telem.Provider>
    </Worker.Provider>
  );
};
