// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import type { Meta } from "@storybook/react";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { Align } from "@/align";
import { TimeRangeChip } from "@/ranger/TimeRangeChip";
import { Text } from "@/text";

const story: Meta<typeof TimeRangeChip> = {
  title: "Time Range Chip",
  component: TimeRangeChip,
};

export const Default = (): ReactElement => {
  const tr1 = TimeStamp.now().spanRange(TimeSpan.seconds(20));
  const tr2 = TimeStamp.now().sub(TimeSpan.days(2)).spanRange(TimeSpan.seconds(20));
  const tr3 = TimeStamp.now().sub(TimeSpan.days(2)).spanRange(TimeSpan.days(1));
  const tr4 = TimeStamp.now().range(TimeStamp.MAX);
  return (
    <Align.Space direction="y">
      <Align.Space direction="x">
        <Text.Text level="small">Nov 9 Bang Bang TPC</Text.Text>
        <TimeRangeChip timeRange={tr1} />
      </Align.Space>
    </Align.Space>
  );
};

export default story;
