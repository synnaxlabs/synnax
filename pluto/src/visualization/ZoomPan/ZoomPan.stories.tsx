import { useState } from "react";

import { ComponentMeta } from "@storybook/react";

import { useZoomPan, ZoomPanMask } from "./ZoomPan";

import { ONE_XY, Box, ZERO_XY } from "@/spatial";

const story: ComponentMeta<typeof ZoomPanMask> = {
  title: "Visualization/ZoomPan",
  component: ZoomPanMask,
};

export const Basic = (): JSX.Element => {
  const [box, setBox] = useState<Box>(new Box(ZERO_XY, ONE_XY));
  const props = useZoomPan({
    threshold: { x: 35, y: 35 },
    onChange: setBox,
    panHotkey: "",
    zoomHotkey: "Shift",
    minZoom: { x: 0.01, y: 0.01 },
    maxZoom: { x: 2, y: 2 },
  });
  return (
    <>
      <ZoomPanMask
        {...props}
        style={{
          position: "relative",
          height: 400,
          width: 400,
          top: 600,
          left: 0,
          border: "1px solid red",
        }}
      />
      <ZoomMiniMap box={box} />
    </>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;

const ZoomMiniMap = ({ box }: { box: Box }): JSX.Element => {
  return (
    <div
      style={{
        top: 200,
        left: 100,
        position: "absolute",
        width: 400,
        height: 400,
        border: "1px solid red",
      }}
    >
      <div
        style={{
          position: "relative",
          top: box.top * 400,
          left: box.left * 400,
          width: box.width * 400,
          height: box.height * 400,
          border: "1px solid blue",
        }}
      />
    </div>
  );
};
