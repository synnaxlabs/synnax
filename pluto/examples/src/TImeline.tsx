// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Timeline, TimeSpan } from "@synnaxlabs/pluto";
import { useState } from "react";

const BARS: Timeline.BarSpec[] = [
  {
    key: "first",
    label: "COPV Test 1",
    color: "#DC136C",
    timeRange: {
      start: TimeSpan.seconds(1),
      end: TimeSpan.seconds(10),
    },
  },
  {
    key: "second",
    label: "COPV Test 2",
    color: "#7AC74F",
    timeRange: {
      start: TimeSpan.seconds(5),
      end: TimeSpan.seconds(15),
    },
  },
];

export const ExampleTimeline = () => {
  const [bars, setBars] = useState(BARS);

  const handleTranslate = (key: string, span: TimeSpan) => {
    setBars((prev) => [
      ...prev.map((b) => {
        if (b.key == key) b.offset = span;
        return b;
      }),
    ]);
  };

  return <Timeline.Timeline onTranslate={handleTranslate} bars={bars} />;
};
