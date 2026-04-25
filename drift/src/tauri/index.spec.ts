// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { buildWindowOptions } from "@/tauri";

describe("buildWindowOptions", () => {
  it("should set the label on the options", () => {
    const options = buildWindowOptions("my-window", {});
    expect(options.label).toBe("my-window");
  });

  it("should always set titleBarStyle to overlay", () => {
    const options = buildWindowOptions("win", {});
    expect(options.titleBarStyle).toBe("overlay");
  });

  it("should always disable drag and drop", () => {
    const options = buildWindowOptions("win", {});
    expect(options.dragDropEnabled).toBe(false);
  });

  it("should map position to x and y", () => {
    const options = buildWindowOptions("win", { position: { x: 100, y: 200 } });
    expect(options.x).toBe(100);
    expect(options.y).toBe(200);
  });

  it("should map size to width and height", () => {
    const options = buildWindowOptions("win", {
      size: { width: 800, height: 600 },
    });
    expect(options.width).toBe(800);
    expect(options.height).toBe(600);
  });

  it("should map minSize to minWidth and minHeight", () => {
    const options = buildWindowOptions("win", {
      minSize: { width: 400, height: 300 },
    });
    expect(options.minWidth).toBe(400);
    expect(options.minHeight).toBe(300);
  });

  it("should map maxSize to maxWidth and maxHeight", () => {
    const options = buildWindowOptions("win", {
      maxSize: { width: 1920, height: 1080 },
    });
    expect(options.maxWidth).toBe(1920);
    expect(options.maxHeight).toBe(1080);
  });

  it("should leave x and y undefined when position is not provided", () => {
    const options = buildWindowOptions("win", {});
    expect(options.x).toBeUndefined();
    expect(options.y).toBeUndefined();
  });

  it("should clamp size dimensions to MIN_DIM (250)", () => {
    const options = buildWindowOptions("win", {
      size: { width: 100, height: 50 },
    });
    expect(options.width).toBe(250);
    expect(options.height).toBe(250);
  });

  it("should clamp maxSize dimensions to MIN_DIM (250)", () => {
    const options = buildWindowOptions("win", {
      maxSize: { width: 100, height: 50 },
    });
    expect(options.maxWidth).toBe(250);
    expect(options.maxHeight).toBe(250);
  });

  it("should not clamp dimensions that are already above MIN_DIM", () => {
    const options = buildWindowOptions("win", {
      size: { width: 500, height: 400 },
      maxSize: { width: 1920, height: 1080 },
    });
    expect(options.width).toBe(500);
    expect(options.height).toBe(400);
    expect(options.maxWidth).toBe(1920);
    expect(options.maxHeight).toBe(1080);
  });

  it("should pass through additional props", () => {
    const options = buildWindowOptions("win", {
      title: "My Window",
      resizable: true,
      visible: false,
      fullscreen: false,
    });
    expect(options.title).toBe("My Window");
    expect(options.resizable).toBe(true);
    expect(options.visible).toBe(false);
    expect(options.fullscreen).toBe(false);
  });
});
