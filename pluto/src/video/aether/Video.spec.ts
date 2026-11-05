// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { telemTest } from "@/telem/aether/testutil";
import { renderAether } from "@/testutil/renderAether";
import { type VideoSourceSpec } from "@/video/aether/telem/types";
import { Video, videoStateZ } from "@/video/aether/Video";
import { type render } from "@/vis/render";
import { canvasTest } from "@/vis/render/testutil";

const REGION = box.construct({ x: 10, y: 20 }, { width: 400, height: 400 });

// A 16:9 picture. The leaf reads only its display size.
const PICTURE = { displayWidth: 160, displayHeight: 90 } as unknown as VideoFrame;

const setup = (overrides: Partial<{ visible: boolean }> = {}) => {
  const source = telemTest.source<VideoFrame | null>(null);
  const spec: VideoSourceSpec = {
    type: telemTest.TestSource.TYPE,
    props: { testId: source.id },
    variant: "source",
    valueType: "video",
  };
  const recorder = canvasTest.record();
  const h = renderAether(Video, {
    state: videoStateZ.parse({ region: REGION, telem: spec, ...overrides }),
    render: recorder,
  });
  // Clears the recordings, then runs the newest render that the leaf asked the loop for.
  const draw = (): render.Cleanup | void => {
    const request = recorder.loopCalls.at(-1)!.args[0] as render.Request;
    recorder.clear();
    return request.render();
  };
  return { h, source, recorder, draw };
};

describe("video/aether/Video", () => {
  it("should stay empty and draw nothing before the source has a picture", () => {
    const { h, recorder, draw } = setup();
    expect(h.component.state.empty).toBe(true);
    draw();
    expect(recorder.lower2d.calls.filter((c) => c.op === "drawImage")).toEqual([]);
  });

  it("should leave the empty state when a picture arrives", () => {
    const { h, source } = setup();
    source.setValue(PICTURE);
    expect(h.component.state.empty).toBe(false);
  });

  it("should draw the picture centered in its region at its aspect ratio", () => {
    const { source, recorder, draw } = setup();
    source.setValue(PICTURE);
    draw();
    const ops = recorder.lower2d.calls.map((c) => c.op);
    expect(ops).toEqual([
      "set:imageSmoothingEnabled",
      "drawImage",
      "set:imageSmoothingEnabled",
    ]);
    expect(recorder.lower2d.calls[1].args).toEqual([PICTURE, 10, 107.5, 400, 225]);
    expect(recorder.lower2d.calls[0].args).toEqual([true]);
    expect(recorder.scissorCalls.map((c) => c.region)).toEqual([REGION]);
  });

  it("should erase its region before the next picture draws", () => {
    const { source, recorder, draw } = setup();
    source.setValue(PICTURE);
    const cleanup = draw();
    source.setValue(PICTURE);
    const next = recorder.loopCalls.at(-1)!.args[0] as render.Request;
    recorder.clear();
    cleanup?.(next);
    expect(recorder.eraseCalls.map((c) => c.region)).toEqual([REGION]);
  });

  it("should ask for no render while hidden", () => {
    const { source, recorder } = setup({ visible: false });
    source.setValue(PICTURE);
    expect(recorder.loopCalls).toEqual([]);
  });

  it("should erase its region when deleted", () => {
    const { h, recorder } = setup();
    recorder.clear();
    h.unmount();
    expect(recorder.eraseCalls.map((c) => c.region)).toEqual([REGION]);
  });
});
