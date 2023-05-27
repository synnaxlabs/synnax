// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { convertRenderV, RenderableRecord, XY } from "@synnaxlabs/x";

import { Space, Typography } from "@/core/std";
import { CSS } from "@/core/css";
import { textWidth } from "@/core/std/Typography/textWidth";
import { useFont } from "@/core/theming";

export interface AnnotationProps {
  position: XY;
  stroke?: string;
  values: RenderableRecord;
}

export const Annotation = ({
  values,
  position,
  stroke = "var(--pluto-gray-m2)",
}: AnnotationProps): ReactElement => {
  return (
    <foreignObject
      x={position.x}
      y={position.y}
      height={annotationHeight(values)}
      width={annotationWidth(values)}
    >
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

export const annotationHeight = (values: RenderableRecord): number =>
  Object.entries(values).length * 20 + 10;

export const annotationWidth = (values: RenderableRecord): number => {
  const font = useFont("small");
  return (
    Math.max(
      ...Object.entries(values).map(([key, value]) =>
        textWidth(`${key}: ${convertRenderV(value) as string}`, font)
      )
    ) + 20
  );
};
