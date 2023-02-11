import { useState } from "react";

import { ComponentMeta } from "@storybook/react";

import { useViewport, ZoomPanSelectMask } from "./ZoomPanSelect";

import { Box, DECIMAL_BOX } from "@/spatial";
import { BoxScale } from "@/spatial/scale";
import { Triggers } from "@/triggers";

const story: ComponentMeta<typeof ZoomPanSelectMask> = {
  title: "Visualization/ZoomPan",
  component: ZoomPanSelectMask,
};

const Basic_ = (): JSX.Element => {
  const [box, setBox] = useState<Box>(DECIMAL_BOX);
  const props = useViewport({
    onChange: ({ box: newBox }) => setBox(newBox),
  });
  return (
    <>
      <ZoomPanSelectMask
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

export const Basic = (): JSX.Element => {
  return (
    <Triggers.Provider>
      <Basic_ />
    </Triggers.Provider>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;

const ZoomMiniMap = ({ box }: { box: Box }): JSX.Element => {
  const scaled = BoxScale.scale(DECIMAL_BOX)
    .scale(new Box(0, 0, 400, 400))
    .box(box)
    .reRoot("topLeft");
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
          top: scaled.top,
          left: scaled.left,
          width: scaled.width,
          height: scaled.height,
          border: "1px solid blue",
        }}
      />
    </div>
  );
};
