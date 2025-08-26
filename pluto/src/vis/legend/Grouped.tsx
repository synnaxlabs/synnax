// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Fragment, type ReactElement, useMemo, useState } from "react";

import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Text } from "@/text";
import { text } from "@/text/core";
import { Theming } from "@/theming";
import { Container, type ContainerProps } from "@/vis/legend/Container";
import { Entries, type EntriesProps } from "@/vis/legend/Entries";

export interface GroupData {
  key: string;
  name: string;
  data: EntriesProps["data"];
}

export interface GroupedProps
  extends Omit<
      ContainerProps,
      "value" | "onChange" | "background" | "draggable" | "gap"
    >,
    Pick<EntriesProps, "background" | "allowVisibleChange" | "onEntryChange"> {
  data: GroupData[];
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
}

export const Grouped = ({
  data,
  background = 1,
  allowVisibleChange = true,
  onEntryChange,
  position,
  onPositionChange,
  ...rest
}: GroupedProps): ReactElement | null => {
  const [pickerVisible, setPickerVisible] = useState(false);
  const font = Theming.useTypography("small").toString();
  const style = useMemo(() => {
    if (data.length === 0) return undefined;
    let width = 0;
    for (const { name } of data) {
      const dims = text.dimensions(name, font);
      width = Math.max(width, dims.width);
    }
    return { width: `${width}px` };
  }, [data, font]);
  if (data.length === 0) return null;
  return (
    <Container
      {...rest}
      draggable={!pickerVisible}
      value={position}
      onChange={onPositionChange}
      gap="small"
      background={background}
    >
      {data.map(({ key, name, data: groupData }, i) => (
        <Fragment key={key}>
          <Flex.Box x>
            <Text.Text level="small" style={style}>
              {name}
            </Text.Text>
            <Flex.Box y grow>
              <Entries
                data={groupData}
                onEntryChange={onEntryChange}
                colorPickerVisible={pickerVisible}
                onColorPickerVisibleChange={setPickerVisible}
                allowVisibleChange={allowVisibleChange}
                background={background}
                entryProps={{ justify: "between", grow: true }}
              />
            </Flex.Box>
          </Flex.Box>
          {i !== data.length - 1 && <Divider.Divider x />}
        </Fragment>
      ))}
    </Container>
  );
};
