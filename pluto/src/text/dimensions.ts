// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type dimensions as core, runtime } from "@synnaxlabs/x";

let canvas: HTMLCanvasElement | null = null;

const getCanvas = (): HTMLCanvasElement => {
  if (canvas == null) canvas = document.createElement("canvas");
  return canvas;
};

export const dimensions = (
  text: string,
  font: string,
  context?: OffscreenCanvasRenderingContext2D | CanvasRenderingContext2D,
): core.Dimensions => {
  if (runtime.RUNTIME === "node") return { width: 0, height: 0 };
  if (context == null)
    context = getCanvas().getContext("2d") as CanvasRenderingContext2D;
  context.font = font;
  const metrics = context.measureText(text);
  return {
    width: Math.trunc(
      Math.abs(metrics.actualBoundingBoxLeft) +
        Math.abs(metrics.actualBoundingBoxRight),
    ),
    height: Math.trunc(
      Math.abs(metrics.actualBoundingBoxAscent) +
        Math.abs(metrics.actualBoundingBoxDescent),
    ),
  };
};
