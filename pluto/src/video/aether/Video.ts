// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type destructor, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { noopVideoSourceSpec } from "@/video/aether/telem/noop";
import { type VideoSource, videoSourceSpecZ } from "@/video/aether/telem/types";
import { clock } from "@/video/clock";
import { render } from "@/vis/render";

export const videoStateZ = z.object({
  region: box.box,
  telem: videoSourceSpecZ.default(noopVideoSourceSpec),
  visible: z.boolean().default(true),
  // empty is true until the view has a picture to draw.
  empty: z.boolean().default(true),
  /** The clock of a video that plays a range. Absent shows live video. */
  clock: clock.clockZ.optional(),
});

const CANVAS: render.Canvas2DVariant = "lower2d";

interface InternalState {
  render: render.Context;
  telem: telem.MemoizedSource<VideoFrame | null, VideoSource>;
  stopListeningTelem?: destructor.Destructor;
  stopListeningRestore?: destructor.Destructor;
  // tick is the animation frame that draws the next picture of a playing clock.
  tick?: number;
}

/** @returns the largest box with the aspect ratio of the picture, centered in region. */
const fit = (picture: VideoFrame, region: box.Box): box.Box => {
  const scale = Math.min(
    box.width(region) / picture.displayWidth,
    box.height(region) / picture.displayHeight,
  );
  const width = picture.displayWidth * scale;
  const height = picture.displayHeight * scale;
  return box.construct(
    box.left(region) + (box.width(region) - width) / 2,
    box.top(region) + (box.height(region) - height) / 2,
    width,
    height,
  );
};

/** Draws the newest decoded picture of a video source into its region. */
export class Video extends aether.Leaf<typeof videoStateZ, InternalState> {
  static readonly TYPE = "video";
  static readonly z = videoStateZ;
  schema = Video.z;
  private renderKey = "";

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    if (this.renderKey === "") this.renderKey = `${this.type}-${this.key}`;
    i.render = render.Context.use(ctx);
    i.stopListeningRestore?.();
    i.stopListeningRestore = i.render.on2DRestore(() => this.requestRender());
    i.telem = telem.useSource(ctx, this.state.telem, i.telem, {
      onStatusChange: status.useOptionalAdder(ctx),
    });
    i.stopListeningTelem?.();
    i.stopListeningTelem = i.telem.onChange(() => this.changed());
    if (this.state.clock?.playing !== true) this.stopTicking();
    else if (i.tick == null) this.startTicking();
    this.changed();
  }

  // Draws on every animation frame while the clock plays.
  private startTicking(): void {
    const tick = (): void => {
      this.internal.tick = requestAnimationFrame(tick);
      this.requestRender();
    };
    this.internal.tick = requestAnimationFrame(tick);
  }

  private stopTicking(): void {
    if (this.internal.tick != null) cancelAnimationFrame(this.internal.tick);
    this.internal.tick = undefined;
  }

  afterDelete(): void {
    const {
      telem,
      render: renderCtx,
      stopListeningRestore,
      stopListeningTelem,
    } = this.internal;
    stopListeningRestore?.();
    stopListeningTelem?.();
    this.stopTicking();
    telem.cleanup?.();
    renderCtx.erase(box.construct(this.state.region), xy.ZERO, CANVAS);
  }

  private value(): VideoFrame | null {
    const { clock: c } = this.state;
    const time = c == null ? undefined : clock.timeAt(c);
    return this.internal.telem.value({ time }).data;
  }

  private changed(): void {
    const empty = this.value() == null;
    if (empty !== this.state.empty) this.setState((s) => ({ ...s, empty }));
    this.requestRender();
  }

  private requestRender(): void {
    if (!this.state.visible && !this.prevState.visible) return;
    this.internal.render.loop.set({
      key: this.renderKey,
      render: () => this.render(),
      priority: "high",
      canvases: [CANVAS],
    });
  }

  private render(): render.Cleanup | undefined {
    const { render: renderCtx } = this.internal;
    const { region, visible } = this.state;
    if (box.areaIsZero(region)) return undefined;
    const erase = (): void => renderCtx.erase(region, xy.ZERO, CANVAS);
    const picture = this.value();
    if (!visible || picture == null) return erase;
    const canvas = renderCtx[CANVAS];
    const clearScissor = renderCtx.scissor(region, xy.ZERO, [CANVAS]);
    canvas.imageSmoothingEnabled = true;
    const target = fit(picture, region);
    canvas.drawImage(
      picture,
      box.left(target),
      box.top(target),
      box.width(target),
      box.height(target),
    );
    canvas.imageSmoothingEnabled = false;
    clearScissor();
    return erase;
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Video.TYPE]: Video };
