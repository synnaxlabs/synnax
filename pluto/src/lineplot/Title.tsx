// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useGridEntry } from "@/lineplot/LinePlot";
import { Text } from "@/text";
import { Theming } from "@/theming";

export type TitleProps = Text.MaybeEditableProps;

export const Title = ({ level = "h2", ...rest }: TitleProps): ReactElement => {
  const key = useUniqueKey();
  const font = Theming.useTypography(level);
  const gridStyle = useGridEntry(
    { key, size: (font.lineHeight + 2) * font.baseSize, loc: "top", order: 10 },
    "LinePlot.Title",
  );
  return (
    <Flex.Box justify="center" align="center" style={gridStyle}>
      <Text.MaybeEditable {...rest} level={level} />
    </Flex.Box>
  );
};
