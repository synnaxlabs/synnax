// Copyright 2025 Synnax Labs, Inc.
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

import { PADDING_STYLE } from "./constants";

const SelectMultiple = () => {
  const [value, setValue] = useState<channel.Key[]>([]);
  return <Channel.SelectMultiple value={value} onChange={setValue} />;
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
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1} full="x">
    <SelectMultiple />
    <SelectSingle />
    <SelectButton />
  </Flex.Box>
);
