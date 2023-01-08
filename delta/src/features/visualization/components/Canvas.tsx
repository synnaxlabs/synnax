// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  PropsWithChildren,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

import { FrameCache, UnexpectedError } from "@synnaxlabs/client";

import { CanvasRenderingContext } from "../render/context";
import { newDefaultRendererRegistry, RenderingContext } from "../render/render";
import { TelemetryClient } from "../telem/client";
import { WebGLBufferCache } from "../telem/glCache";
import { FrameRetriever } from "../telem/retriever";

import { useClusterClient } from "@/features/cluster";

export interface CanvasContextValue {
  ctx: RenderingContext | null;
}

const CanvasContext = createContext<CanvasContextValue | null>(null);

export const useRenderingContext = (): RenderingContext | null => {
  const ctx = useContext(CanvasContext);
  if (ctx == null) return null;
  return ctx.ctx;
};

export interface CanvasProps extends PropsWithChildren<any> {}

export const Canvas = ({ children }: CanvasProps): JSX.Element => {
  const [ctx, setCtx] = useState<RenderingContext | null>(null);
  const ref = useRef<HTMLCanvasElement | null>(null);
  const glRef = useRef<WebGLRenderingContext | null>(null);

  const client = useClusterClient();

  useEffect(() => refProxy(ref.current), [client]);

  const refProxy = (e: HTMLCanvasElement | null): void => {
    if (client == null || e == null || ctx !== null) return;
    if (glRef.current === null) {
      glRef.current = e.getContext("webgl", { preserveDrawingBuffer: true });
    }
    const gl = glRef.current;
    if (gl == null) throw new UnexpectedError("failed to initialize WebGL context");
    const reg = newDefaultRendererRegistry();
    setCtx(
      new CanvasRenderingContext(
        e,
        gl,
        reg,
        new TelemetryClient(
          new WebGLBufferCache(gl),
          new FrameRetriever(client),
          new FrameCache()
        )
      )
    );
    ref.current = e;
  };

  return (
    <CanvasContext.Provider value={{ ctx }}>
      <canvas
        ref={refProxy}
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
