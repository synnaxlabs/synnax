// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, PropsWithChildren, useContext, useRef } from "react";

import { RenderingEngine, RenderRequest } from "../engine/engine";

export interface CanvasContextValue {
  render: RenderF;
}

type RenderF = (req: RenderRequest) => void;

const CanvasContext = createContext<CanvasContextValue | null>(null);

export const useRenderer = (): RenderF => {
  const ctx = useContext(CanvasContext);
  if (ctx == null) throw new Error("Canvas context not found");
  return ctx.render;
};

export interface CanvasProps extends PropsWithChildren<any> {}

export const Canvas = ({ children }: CanvasProps): JSX.Element => {
  const ref = useRef<RenderingEngine | null>(null);
  const { current: engine } = ref;

  const render = (req: RenderRequest): void => engine?.render(req);

  return (
    <CanvasContext.Provider value={{ render }}>
      <canvas
        ref={(e) => {
          ref.current = new RenderingEngine(e as HTMLCanvasElement);
        }}
        style={{
          width: "100%",
          height: "100%",
        }}
      />
      {children}
    </CanvasContext.Provider>
  );
};
