// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { useState } from "react";

import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Select } from "@/select";
import { Text } from "@/text";

import { SubcategorySection } from "./SubcategorySection";

const SelectMultiple = () => {
  const [value, setValue] = useState<channel.Key[]>([]);
  return (
    <Channel.SelectMultiple
      value={value}
      onChange={setValue}
      triggerProps={{ variant: "text" }}
    />
  );
};

const SelectSingle = () => {
  const [value, setValue] = useState<channel.Key | undefined>(undefined);
  return <Channel.SelectSingle value={value} onChange={setValue} />;
};

const SelectButton = () => {
  const [value, setValue] = useState<string>("");
  return (
    <Select.Buttons
      keys={["x-center", "y-center", "x-left", "y-left"]}
      value={value}
      onChange={setValue}
    >
      <Select.Button key="x-center" itemKey="x-center">
        <Icon.Align.XCenter />
      </Select.Button>
      <Select.Button key="y-center" itemKey="y-center">
        <Icon.Align.YCenter />
      </Select.Button>
      <Select.Button key="x-left" itemKey="x-left">
        <Icon.Align.Left />
      </Select.Button>
      <Select.Button key="y-left" itemKey="y-left">
        <Icon.Align.Top />
      </Select.Button>
    </Select.Buttons>
  );
};

export const SelectShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Channel Selection"
        description="Dropdown selectors for single and multiple channel selection with search capabilities"
      >
        <Flex.Box y gap="huge">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Multiple Selection
            </Text.Text>
            <SelectMultiple />
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Single Selection
            </Text.Text>
            <SelectSingle />
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Button Selection"
        description="Toggle button groups for selecting from predefined options with visual icons"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Icon Button Group
          </Text.Text>
          <SelectButton />
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Icon Selection"
        description="Select from a list of icons"
      >
        <SelectIconShowcase />
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);

export const SelectIconShowcase = () => {
  const [value, setValue] = useState<string>("");
  return (
    <Select.Static
      resourceName="alignment"
      data={[
        { key: "x-center", icon: <Icon.Align.XCenter />, name: "X Center" },
        { key: "y-center", icon: <Icon.Align.YCenter />, name: "Y Center" },
        { key: "x-left", icon: <Icon.Align.Left />, name: "X Left" },
        { key: "y-left", icon: <Icon.Align.Top />, name: "Y Left" },
      ]}
      value={value}
      allowNone
      variant="floating"
      onChange={setValue}
      icon={<Icon.Align.XCenter />}
      triggerProps={{ iconOnly: true }}
    />
  );
};
