// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, PropsWithChildren, useContext, useRef } from "react";

import { UnexpectedError } from "@synnaxlabs/client";

import { RenderingEngine, RenderRequest } from "../engine/engine";
import { RENDERER_REGISTRY } from "../render/registry";
import { TelemetryClient } from "../telem/client";
import { WebGLBufferCache } from "../telem/glCache";
import { FrameRetriever } from "../telem/retriever";

import { useClusterClient } from "@/features/cluster";

export interface CanvasContextValue {
  render: RenderF;
}

type RenderF = (req: RenderRequest) => Promise<void>;

const CanvasContext = createContext<CanvasContextValue | null>(null);

export const useRenderer = (): RenderF => {
  const ctx = useContext(CanvasContext);
  if (ctx == null) throw new Error("Canvas context not found");
  return ctx.render;
};

export interface CanvasProps extends PropsWithChildren<any> {}

export const Canvas = ({ children }: CanvasProps): JSX.Element => {
  const ref = useRef<RenderingEngine | null>(null);

  const client = useClusterClient();

  const render = async (req: RenderRequest): Promise<void> => {
    await ref.current?.render(req);
  };

  const newEngine = (e: HTMLCanvasElement): void => {
    if (client == null || e == null) return;
    const gl = e.getContext("webgl", {
      preserveDrawingBuffer: true,
    });
    if (gl == null) throw new UnexpectedError("failed to initialize WebGL context");
    ref.current = new RenderingEngine(
      e,
      gl,
      RENDERER_REGISTRY,
      new TelemetryClient(new WebGLBufferCache(gl), new FrameRetriever(client))
    );
  };

  return (
    <CanvasContext.Provider value={{ render }}>
      <canvas
        ref={newEngine}
        style={{
          position: "absolute",
          width: "100%",
          height: "100%",
        }}
      />
      {children}
    </CanvasContext.Provider>
  );
};
