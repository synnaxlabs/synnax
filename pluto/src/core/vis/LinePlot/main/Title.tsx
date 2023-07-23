// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { Space, Text, TextMaybeEditableProps, TypographyLevel } from "@/core/std";
import { Theming } from "@/core/theming";
import { useGridPosition } from "@/core/vis/LinePlot/main/LinePlot";

export type TitleProps<L extends TypographyLevel = "h2"> = TextMaybeEditableProps<L>;

export const Title = <L extends TypographyLevel = "h2">({
  level = "h2" as TitleProps<L>["level"],
  ...props
}: TitleProps<L>): ReactElement => {
  const key = useUniqueKey();
  const font = Theming.useTypography(level);
  const gridStyle = useGridPosition(
    {
      key,
      size: (font.lineHeight + 0.5) * font.baseSize,
      loc: "top",
      order: "first",
    },
    "Title"
  );
  return (
    <Space justify="center" align="center" style={gridStyle}>
      {/* @ts-expect-error */}
      <Text.MaybeEditable<L> {...props} level={level} />
    </Space>
  );
};
