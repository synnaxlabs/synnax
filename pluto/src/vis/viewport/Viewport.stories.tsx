// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { ComponentMeta } from "@storybook/react";
import { Box, DECIMAL_BOX, BoxScale } from "@synnaxlabs/x";

import { Viewport } from ".";

import { Triggers } from "@/triggers";

const story: ComponentMeta<typeof Viewport.Mask> = {
  title: "Visualization/ZoomPan",
  component: Viewport.Mask,
};

const Basic_ = (): JSX.Element => {
  const [box, setBox] = useState<Box>(DECIMAL_BOX);
  const props = Viewport.use({
    onChange: ({ box: newBox }) => setBox(newBox),
  });
  return (
    <>
      <Viewport.Mask
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
