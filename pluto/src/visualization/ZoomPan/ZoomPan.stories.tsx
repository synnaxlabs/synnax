import { ComponentMeta } from "@storybook/react";

import { useZoomPan, ZoomPanMask } from "./ZoomPan";

const story: ComponentMeta<typeof ZoomPanMask> = {
  title: "Visualization/ZoomPan",
  component: ZoomPanMask,
};

export const Basic = (): JSX.Element => {
  const props = useZoomPan({ threshold: { x: 35, y: 35 } });
  return (
    <ZoomPanMask
      {...props}
      style={{
        position: "relative",
        height: "50%",
        width: "50%",
        top: 200,
        left: 200,
        border: "1px solid red",
      }}
    />
  );
};

export default story;
