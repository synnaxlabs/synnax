// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const canvas = document.createElement("canvas");

export const textWidth = (text: string, font: string): number => {
  const context = canvas.getContext("2d") as CanvasRenderingContext2D;
  context.font = font;
  const metrics = context.measureText(text);
  return Math.trunc(metrics.width);
};
