// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { Synnax, SynnaxProps } from "@synnaxlabs/client";
import { Box, BoxT } from "@synnaxlabs/x";

import { newDefaultRendererRegistry } from "../core/vis/render/registry";

import { Theme } from "@/core/theming";
import { RenderContext } from "@/core/vis/render/RenderContext";
import { Client } from "@/telem/client";
import { LineVis, LineVisState } from "@/vis/line/core/line";

class CanvasWorker {
  telem: Client;
  glContext: RenderContext;
  plots: Map<String, LineVis>;
  theme: Theme;

  constructor(telem: Client, glContext: RenderContext, theme: Theme) {
    this.telem = telem;
    this.glContext = glContext;
    this.plots = new Map();
    this.theme = theme;
  }

  async updateConn(props: SynnaxProps): Promise<void> {
    const client = new Synnax(props);
    await client.connectivity.check();
    this.telem = new Client(client, this.glContext.gl);
    await this.render();
  }

  async updateTheme(theme: Theme): Promise<void> {
    this.theme = theme;
    await this.render();
  }

  async resize(
    box: Box,
    dpr: number,
    viewport: [number, number] | null
  ): Promise<void> {
    this.glContext.updateCanvasRegion(box, dpr);
    if (viewport != null) {
      this.glContext.gl.viewport(0, 0, ...viewport);
    }
    await this.render();
  }

  async updateVis(state: LineVisState): Promise<void> {
    const key = state.key;
    let plot = this.plots.get(key);
    if (plot == null) {
      plot = new LineVis(key);
      this.plots.set(key, plot);
    }
    plot.update(state);
    await plot.build({
      theme: this.theme,
      client: this.telem,
    });
    await plot.cleanup({ gl: this.glContext });
    await plot.render({ gl: this.glContext });
  }

  private async render(): Promise<void> {
    for (const plot of this.plots.values()) {
      await plot.build({ theme: this.theme, client: this.telem });
      await plot.render({ gl: this.glContext });
    }
  }
}

export interface WorkerBootstrap extends Resize {
  connParams: SynnaxProps;
  canvas: OffscreenCanvas;
  theme: Theme;
}

export interface Resize {
  box: BoxT;
  dpr: number;
  viewport: [number, number] | null;
}

export interface WorkerMessage {
  type: "bootstrap" | "render" | "updateConn" | "resize" | "theme";
  data: WorkerBootstrap | LineVisState | SynnaxProps | Resize | Theme;
}

let worker: CanvasWorker | null = null;

onmessage = async (e: MessageEvent<WorkerMessage>): Promise<void> => {
  const msg = e.data;
  if (msg.type === "bootstrap") {
    const data = msg.data as WorkerBootstrap;
    const ctx = new RenderContext(
      data.canvas,
      newDefaultRendererRegistry(),
      new Box(data.box),
      data.dpr
    );
    worker = new CanvasWorker(
      new Client(new Synnax(data.connParams), ctx.gl),
      ctx,
      data.theme
    );
    postMessage("ready");
  }

  if (worker == null) throw new Error("Worker not initialized");

  if (msg.type === "theme") {
    await worker.updateTheme(msg.data as Theme);
  } else if (msg.type === "bootstrap") {
    await worker.updateConn(msg.data as SynnaxProps);
  } else if (msg.type === "render") {
    await worker.updateVis(msg.data as LineVisState);
  } else if (msg.type === "resize") {
    const data = msg.data as Resize;
    await worker.resize(new Box(data.box), data.dpr, data.viewport);
  } else if (msg.type === "updateConn") {
    await worker.updateConn(msg.data as SynnaxProps);
  }
};
