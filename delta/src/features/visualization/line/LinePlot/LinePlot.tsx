// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSSProperties, useEffect, useState } from "react";

import { Synnax } from "@synnaxlabs/client";
import { Autosize, useDrag, UseSizeReturn } from "@synnaxlabs/pluto";

import { useRenderer } from "../../components/Canvas";
import { CSSBox, PointBox } from "../../types/spatial";
import { Visualization } from "../../types";

import "./LinePlot.css";
import { sort } from "d3";

export interface LinePlotProps {
  visualization: SugaredLinePlotVisualization;
  onChange: (vis: Visualization) => void;
  client: Synnax;
  resizeDebounce: number;
}

export const LinePlot = (): JSX.Element => {
  return (
    <div className="delta-line-plot__container">
      <Autosize className="delta-line-plot__plot__container" debounce={100}>
        {({ width, height, left, top }) => (
          <>
            <CorePlot width={width} height={height} left={left} top={top} />
          </>
        )}
      </Autosize>
    </div>
  );
};

const count = 1e4 * 5;
const xData = Float32Array.from({ length: count }, (_, i) => i / count);
const yData = Float32Array.from({ length: count }, (_, i) => {
  // generate a step function
  // return Math.sin(i / 500000);
  const x = i / count;
  if (x < 0.25) {
    return 0;
  } else if (x < 0.5) {
    return 0.25;
  } else if (x < 0.75) {
    return 0.5;
  } else {
    return 0.75;
  }
});

const sorted = sort(yData);
const min = sorted[0];
const max = sorted[sorted.length - 1];

const CorePlot = ({ width, height, left, top }: UseSizeReturn): JSX.Element => {
  const render = useRenderer();

  useEffect(() => {
    render({
      box: {
        width,
        height,
        left,
        top,
      },
      lines: [
        {
          x: xData,
          y: yData,
          scale: {
            x: 1,
            y: 1 / ((max - min) * 1.1),
          },
          offset: {
            x: 0,
            y: 0.1,
          },
          color: [Math.random(), Math.random(), Math.random(), 1],
        },
      ],
    });
  }, [width, height, left, top]);

  const [zoomMask, setZoomMask] = useState<PointBox | null>(null);

  const onZoomDrag = (e: MouseEvent): void =>
    setZoomMask((prev) => ({
      ...(prev ?? { one: { x: e.clientX, y: e.clientY } }),
      two: {
        x: e.clientX,
        y: e.clientY,
      },
    }));

  const dragProps = useDrag({
    onMove: onZoomDrag,
    onEnd: () => {
      setZoomMask(null);
    },
  });

  const zoomMaskStyle: CSSProperties | null = {
    position: "fixed",
    backgroundColor: "rgba(0, 0, 0, 0.2)",
  };

  if (zoomMask != null) {
    zoomMaskStyle.width = Math.abs(zoomMask.one.x - zoomMask.two.x);
    zoomMaskStyle.height = Math.abs(zoomMask.one.y - zoomMask.two.y);
    zoomMaskStyle.left = Math.min(zoomMask.one.x, zoomMask.two.x);
    zoomMaskStyle.top = Math.min(zoomMask.one.y, zoomMask.two.y);
    if (zoomMaskStyle.height < 35) {
      zoomMaskStyle.height = height;
      zoomMaskStyle.top = top;
    } else if (zoomMaskStyle.width < 35) {
      zoomMaskStyle.width = width;
      zoomMaskStyle.left = left;
    }
  }

  return (
    <>
      <div style={{ width, height }} onMouseDown={dragProps.onDragStart} />;
      {zoomMaskStyle != null && <div style={zoomMaskStyle} />}
    </>
  );
};
