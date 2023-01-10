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

import { CanvasRenderingContext } from "../gl/context";
import { newDefaultRendererRegistry, RenderingContext } from "../gl/render";
import { TelemetryClient } from "../telem/client";
import { WebGLBufferCache } from "../telem/glCache";

import { useClusterClient } from "@/features/cluster";

import "./Canvas.css";

export interface VisCanvasContextValue {
  ctx: RenderingContext | null;
}

const VisCanvasContext = createContext<VisCanvasContextValue | null>(null);

export const useVisCanvas = (): RenderingContext | null => {
  const ctx = useContext(VisCanvasContext);
  if (ctx == null) return null;
  return ctx.ctx;
};

export interface VisCanvasProps extends PropsWithChildren<any> {}

export const VisCanvas = ({ children }: VisCanvasProps): JSX.Element => {
  const [ctx, setCtx] = useState<RenderingContext | null>(null);
  const ref = useRef<HTMLCanvasElement | null>(null);
  const glRef = useRef<WebGLRenderingContext | null>(null);

  const client = useClusterClient();

  useEffect(() => callbackRef(ref.current), [client]);

  const callbackRef = (e: HTMLCanvasElement | null): void => {
    if (client == null || e == null || ctx !== null) return;
    if (glRef.current === null)
      glRef.current = e.getContext("webgl", { preserveDrawingBuffer: true });
    const gl = glRef.current;
    if (gl == null) throw new UnexpectedError("failed to initialize WebGL context");
    const reg = newDefaultRendererRegistry();
    setCtx(
      new CanvasRenderingContext(
        e,
        gl,
        reg,
        new TelemetryClient(new WebGLBufferCache(gl), client, new FrameCache())
      )
    );
    ref.current = e;
  };

  return (
    <VisCanvasContext.Provider value={{ ctx }}>
      <canvas ref={callbackRef} className="delta-visualization__canvas" />
      {children}
    </VisCanvasContext.Provider>
  );
};
