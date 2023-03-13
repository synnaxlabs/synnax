// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { convertRenderV, RenderableValue, XY } from "@synnaxlabs/x";

import { Space, Typography } from "@/core";
import { textWidth } from "@/core/Typography/textWidth";
import { CSS } from "@/css";
import { useFont } from "@/theming";

export interface AnnotationProps {
  position: XY;
  stroke?: string;
  values: Record<string, RenderableValue>;
}

export const Annotation = ({
  values,
  position,
  stroke = "var(--pluto-gray-m2)",
}: AnnotationProps): JSX.Element => {
  const textItems = Object.entries(values).map(
    ([key, value]) => `${key}: ${convertRenderV(value) as string}`
  );
  const font = useFont("small");
  const maxWidth = Math.max(...textItems.map((t) => textWidth(t, font)));
  return (
    <foreignObject x={position.x} y={position.y} height="500" width={maxWidth + 20}>
      <Space
        direction="y"
        style={{
          backgroundColor: "var(--pluto-gray-m3)",
          padding: "1rem",
          borderColor: stroke,
        }}
        size="small"
        className={CSS(CSS.B("annotation"), CSS.bordered(), CSS.rounded())}
      >
        {Object.entries(values).map(([key, value]) => (
          <Typography.Text key={key} level="small">
            {`${key}: ${convertRenderV(value) as string}`}
          </Typography.Text>
        ))}
      </Space>
    </foreignObject>
  );
};
