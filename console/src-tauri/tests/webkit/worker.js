// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Draws each frame the way Pluto does: WebGL traces between two 2D layers, with text
// copied from a glyph atlas. Posts a frame to the page as bitmaps only after the page
// acknowledges the last one.
onmessage = ({ data: { width, height, port } }) => {
  const gl = new OffscreenCanvas(width, height).getContext("webgl2", {
    preserveDrawingBuffer: false,
    antialias: false,
    powerPreference: "high-performance",
    stencil: false,
    depth: false,
  });
  if (gl == null) {
    postMessage("no webgl");
    return;
  }
  const framebuffer = gl.createFramebuffer();
  const renderbuffer = gl.createRenderbuffer();
  gl.bindRenderbuffer(gl.RENDERBUFFER, renderbuffer);
  const samples = Math.min(4, gl.getParameter(gl.MAX_SAMPLES));
  gl.renderbufferStorageMultisample(gl.RENDERBUFFER, samples, gl.RGBA8, width, height);
  gl.bindFramebuffer(gl.FRAMEBUFFER, framebuffer);
  gl.framebufferRenderbuffer(
    gl.FRAMEBUFFER,
    gl.COLOR_ATTACHMENT0,
    gl.RENDERBUFFER,
    renderbuffer,
  );

  const atlas = new OffscreenCanvas(256, 256).getContext("2d");
  atlas.font = "20px sans-serif";
  atlas.fillStyle = "#fff";
  for (let i = 0; i < 64; i++)
    atlas.fillText(String.fromCharCode(48 + i), (i % 8) * 32, 24 + (i >> 3) * 32);
  const layers = {
    lower2d: new OffscreenCanvas(width, height).getContext("2d"),
    upper2d: new OffscreenCanvas(width, height).getContext("2d"),
  };
  const copy = new OffscreenCanvas(width, height).getContext("2d");

  let frames = 0;
  let inFlight = false;
  port.onmessage = () => (inFlight = false);
  const present = () => {
    gl.bindFramebuffer(gl.READ_FRAMEBUFFER, framebuffer);
    gl.bindFramebuffer(gl.DRAW_FRAMEBUFFER, null);
    gl.blitFramebuffer(
      0,
      0,
      width,
      height,
      0,
      0,
      width,
      height,
      gl.COLOR_BUFFER_BIT,
      gl.NEAREST,
    );
    gl.bindFramebuffer(gl.FRAMEBUFFER, framebuffer);
    const frame = { gl: gl.canvas.transferToImageBitmap() };
    copy.globalCompositeOperation = "copy";
    Object.entries(layers).forEach(([variant, ctx]) => {
      copy.drawImage(ctx.canvas, 0, 0);
      frame[variant] = copy.canvas.transferToImageBitmap();
    });
    port.postMessage(frame, Object.values(frame));
    inFlight = true;
    if (++frames % 30 === 0) postMessage(`frames ${frames}`);
  };

  const draw = (now) => {
    gl.clearColor(0, 0, Math.random(), 1);
    gl.clear(gl.COLOR_BUFFER_BIT);
    gl.enable(gl.SCISSOR_TEST);
    for (let i = 0; i < 50; i++) {
      gl.scissor((i * 37 + now / 5) % width, (i * 53) % height, 24, 4);
      gl.clearColor(1, i / 50, 0, 1);
      gl.clear(gl.COLOR_BUFFER_BIT);
    }
    gl.disable(gl.SCISSOR_TEST);
    gl.flush();
    Object.values(layers).forEach((ctx, k) => {
      ctx.clearRect(0, 0, width, height);
      ctx.fillStyle = k === 0 ? "#3774d0" : "#f5a623";
      for (let i = 0; i < 200; i++) {
        const x = (i * 37 + now / 5) % width;
        const y = (i * 53) % height;
        ctx.fillRect(x, y, 24, 12);
        ctx.drawImage(atlas.canvas, (i % 8) * 32, 0, 32, 32, x, y + 12, 32, 32);
      }
    });

    if (!inFlight) present();
    requestAnimationFrame(draw);
  };
  requestAnimationFrame(draw);
};
