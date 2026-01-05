// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type dimensions as core, runtime } from "@synnaxlabs/x";

let canvas: HTMLCanvasElement | null = null;

/**
 * Gets or creates a singleton canvas element for text measurement.
 * This canvas is reused across measurements to improve performance.
 * @returns {HTMLCanvasElement} A canvas element for text measurement
 */
const getCanvas = (): HTMLCanvasElement => {
  canvas ??= document.createElement("canvas");
  return canvas;
};

/**
 * Calculates the dimensions (width and height) of a text string when rendered with a specific font.
 * This function uses the canvas API to measure text dimensions accurately.
 *
 * @param {string} text - The text string to measure
 * @param {string} font - The CSS font string to use for measurement (e.g. "16px Arial")
 * @param {OffscreenCanvasRenderingContext2D | CanvasRenderingContext2D} [context] - Optional rendering context to use for measurement
 * @returns {core.Dimensions} An object containing the width and height of the text in pixels
 * @example
 * const dims = dimensions("Hello World", "16px Arial");
 * console.log(dims.width, dims.height);
 */
export const dimensions = (
  text: string,
  font: string,
  context?: OffscreenCanvasRenderingContext2D | CanvasRenderingContext2D,
): core.Dimensions => {
  if (runtime.RUNTIME === "node") return { width: 0, height: 0 };
  context ??= getCanvas().getContext("2d") as CanvasRenderingContext2D;
  context.font = font;
  const metrics = context.measureText(text);
  return dimensionsFromMetrics(metrics);
};

/**
 * Converts TextMetrics from the canvas API into a simplified dimensions object.
 * This function handles the calculation of actual text dimensions by considering
 * the bounding box metrics provided by the canvas API.
 *
 * @param {TextMetrics} metrics - The TextMetrics object from canvas.measureText()
 * @returns {core.Dimensions} An object containing the width and height of the text in pixels
 */
export const dimensionsFromMetrics = (metrics: TextMetrics): core.Dimensions => ({
  width: Math.trunc(
    Math.abs(metrics.actualBoundingBoxLeft) + Math.abs(metrics.actualBoundingBoxRight),
  ),
  height: Math.trunc(
    Math.abs(metrics.actualBoundingBoxAscent) +
      Math.abs(metrics.actualBoundingBoxDescent),
  ),
});
