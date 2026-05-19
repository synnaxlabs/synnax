// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/legend/Grouped.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { Fragment, type ReactElement, useState } from "react";

import { Container, type ContainerProps } from "@/legend/Container";
import { Entries, type EntriesProps } from "@/legend/Entries";

export interface GroupData {
  key: string;
  name: string;
  data: EntriesProps["data"];
}

export interface GroupedProps
  extends
    Omit<ContainerProps, "value" | "onChange" | "background" | "draggable" | "gap">,
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
  if (data.length === 0) return null;
  return (
    <Container
      {...rest}
      draggable={!pickerVisible}
      value={position}
      onChange={onPositionChange}
      gap="small"
      background={background}
      className={CSS.B("legend-grouped")}
    >
      {data.map(({ key, name, data: groupData }, i) => (
        <Fragment key={key}>
          <Text.Text
            level="small"
            color={9}
            weight={500}
            className={CSS.B("legend-name")}
          >
            {name}
          </Text.Text>
          <Flex.Box y grow className={CSS.B("legend-entries")} empty>
            <Entries
              data={groupData}
              onEntryChange={onEntryChange}
              colorPickerVisible={pickerVisible}
              onColorPickerVisibleChange={setPickerVisible}
              allowVisibleChange={allowVisibleChange}
              background={background}
              entryProps={entryProps}
            />
          </Flex.Box>
          {i !== data.length - 1 && <div className={CSS.B("legend-divider")} />}
        </Fragment>
      ))}
    </Container>
  );
};

const entryProps = { justify: "between", grow: true } as const;
