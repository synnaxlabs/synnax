// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useState, useTransition } from "react";

import {
  GLLines,
  Axis,
  Viewport,
  UseViewportHandler,
  Menu as PMenu,
  useResize,
  Status,
  Rule,
} from "@synnaxlabs/pluto";
import { Box, DECIMAL_BOX, TimeStamp, ZERO_BOX } from "@synnaxlabs/x";

import { X_AXIS_KEYS, Y_AXIS_KEYS } from "../../../../vis/types";
import { useTelemetryClient } from "../../../telem/TelemetryContext";
import { LineSVis } from "../types";

import { Axes, AxesState } from "./axes";
import { BoundsState, Bounds } from "./bounds";
import { ContextMenu } from "./ContextMenu";
import { Data, DataState } from "./data";
import { GL, GLState } from "./gl";
import { Scales, ScalesState } from "./scale";

import { useSelectTheme } from "@/features/layout";
import { useAsyncEffect } from "@/hooks";

import "./LinePlot.css";

export interface LinePlotProps {
  vis: LineSVis;
  onChange: (vis: LineSVis) => void;
  resizeDebounce: number;
}

interface HoverState {
  cursor: XY;
  box: Box;
}

export const LinePlot = ({ vis }: LinePlotProps): JSX.Element => {
  const theme = useSelectTheme();
  const client = useTelemetryClient();

  const [data, setData] = useState<DataState>(Data.initial());
  const [scale, setScale] = useState<ScalesState>(Scales.initial());
  const [axes, setAxes] = useState<AxesState>(Axes.initial());
  const [gl, setGL] = useState<GLState>(GL.initial());

  const [container, setContainer] = useState<Box>(ZERO_BOX);

  const [zoom, setZoom] = useState<Box>(DECIMAL_BOX);
  const [selection, setSelection] = useState<Box | null>(null);
  const [hover, setHover] = useState<HoverState | null>(null);

  const [, startDraw] = useTransition();
  const [tick, setTick] = useState(0);

  const valid = isValid(vis);
  const live = isLive(vis);

  useEffect(() => {
    if (!live) return;
    const i = setInterval(() => {
      setTick((t) => t + 1);
    }, 2000);
    return () => clearInterval(i);
  }, [live]);

  useAsyncEffect(async () => {
    if (client == null || !valid) return setData(Data.initial());
    setData(await Data.fetch(vis, client, live));
  }, [live, vis, tick, client]);

  useEffect(() => {
    let scales: ScalesState, axes: AxesState, gl: GLState, bounds: BoundsState;
    if (theme == null || data == null || data.error != null) {
      scales = Scales.initial();
      axes = Axes.initial();
      gl = GL.initial();
      bounds = Bounds.initial();
    } else {
      bounds = Bounds.build(data.data, 15);
      scales = Scales.build(bounds, zoom);
      axes = Axes.build(container, scales);
      gl = GL.build(container, data.data, scales, axes, theme);
    }
    startDraw(() => {
      setGL(gl);
      setScale(scales);
      setAxes(axes);
    });
  }, [zoom, theme, container, data]);

  const menuProps = PMenu.useContextMenu();

  const handleViewport: UseViewportHandler = useCallback((props) => {
    const { box, mode, cursor } = props;
    if (mode === "hover") {
      return setHover({ cursor, box });
    }
    if (mode === "select") {
      setSelection(box);
      return menuProps.open(cursor);
    }
    setSelection(null);
    setZoom(box);
  }, []);

  const viewportProps = Viewport.use({
    onChange: handleViewport,
  });

  const handleResize = useCallback((box: Box) => setContainer(box), [setContainer]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  if (data.error != null)
    return (
      <Status.Text.Centered level="h4" variant="error" hideIcon>
        {data.error.message}
      </Status.Text.Centered>
    );
  if (valid && data == null)
    return (
      <Status.Text.Centered level="h4" variant="loading" hideIcon>
        Loading...
      </Status.Text.Centered>
    );
  if (!valid)
    return (
      <Status.Text.Centered level="h4" variant="disabled" hideIcon>
        Invalid Visualization
      </Status.Text.Centered>
    );

  if (valid && Object.values(data).flat().length === 0)
    return (
      <Status.Text.Centered level="h4" variant="disabled" hideIcon>
        No Data Found
      </Status.Text.Centered>
    );

  return (
    <PMenu.ContextMenu
      className="delta-line-plot__container"
      {...menuProps}
      menu={() => (
        <ContextMenu scale={scale.decimal.forward.x1} selection={selection} />
      )}
    >
      <div className="delta-line-plot__plot" ref={resizeRef}>
        <Viewport.Mask
          style={{ position: "absolute", ...axes.innerBox.css }}
          {...viewportProps}
        />
        <GLLines lines={gl.lines} box={gl.box} />
        <svg className="delta-line-plot__svg">
          {Object.entries(axes.axes).map(([key, axis]) => (
            <Axis key={key} {...axis} />
          ))}
          <Tooltip
            hover={hover}
            scales={scale}
            data={data}
            container={container}
            axes={axes}
          />
        </svg>
      </div>
    </PMenu.ContextMenu>
  );
};

const isValid = (vis: LineSVis): boolean => {
  const hasRanges = X_AXIS_KEYS.some((key) => {
    const v = vis.ranges[key];
    return v?.length > 0;
  });
  const hasXAxis = X_AXIS_KEYS.some((key) => {
    const v = vis.channels[key];
    return v != null && v.length > 0;
  });
  const hasYAxis = Y_AXIS_KEYS.some((key) => {
    const v = vis.channels[key];
    return v?.length > 0;
  });
  return hasRanges && hasXAxis && hasYAxis;
};

const isLive = (vis: LineSVis): boolean => {
  const now = TimeStamp.now();
  return [...vis.ranges.x1, ...vis.ranges.x2].some((r) =>
    new TimeStamp(r.end).after(now)
  );
};

interface TooltipProps {
  container: Box;
  hover: HoverState | null;
  scales: ScalesState;
  data: DataState;
  axes: AxesState;
}

export const Tooltip = ({
  hover,
  scales,
  axes,
  data,
  container,
}: TooltipProps): JSX.Element => {
  if (hover == null) return <></>;
  try {
    const scale = scales.decimal.reverse.x1;
    const x = scale?.pos(hover.box.left);
    if (data.data.x1.length === 0) return <></>;
    if (data.data.x1[0].arrays.length === 0) return <></>;
    const index = data.data.x1[0].arrays[0].binarySearch(BigInt(x));
    const timestamp = data.data.x1[0].arrays[0].data[index];
    if (index === -1) return <></>;
    const y = data.data.y1[0].arrays[0].data[index];
    const xCoord = scales.normal.forward.x1;
    const left = xCoord.pos(Number(timestamp)) * container.width;
    console.log(left);
    return (
      <Rule
        direction="y"
        position={left}
        size={{
          upper: axes.innerBox.height + axes.innerBox.top,
          lower: axes.innerBox.top,
        }}
      />
    );
  } catch (e) {
    console.error(e);
    return <></>;
  }
};

