// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Local review material only. Do not publish or commit before partner approval.
// Original Synnax LinkedIn language: Matt, Sep 30, 2026; approved by Rebecca,
// Oct 15, 2026, in “Intro Rigetti <> Synnax”. That approval does not establish
// approval of the blog presentation, new headlines, artwork, or logo placement.
export const announcement = [
  "We’re excited to welcome Rigetti Computing as a Synnax customer.",
  "Rigetti builds superconducting quantum computers that operate in fridges at near absolute zero temperatures.",
  "Running experiments on new quantum processors requires delicate operations for warming and cooling these fridges.",
  "Rigetti's engineers are adopting Synnax to accelerate fridge operations. Together, we're optimizing automated warmup and cool down procedures, improving operational visibility, and simulating critical workflows before deployment.",
  "All with one goal: faster turnaround to run more experiments on novel quantum technologies.",
];

export const directions = [
  {
    key: "instrument",
    number: "01",
    name: "The instrument",
    description:
      "A single refrigerator, drawn with the precision of the hardware it operates. Blue and teal stay in the details.",
    note: "Illustration · Silver, Synnax blue, Rigetti teal",
    banner: "mesh",
  },
  {
    key: "mesh",
    number: "02",
    name: "Common ground",
    description:
      "Both palettes meet in a quiet mesh. A broad, matte color field with just a trace of magenta and yellow.",
    note: "Mesh gradient · Combined brand palettes",
    banner: "mesh",
  },
  {
    key: "cycle",
    number: "03",
    name: "The next experiment",
    description:
      "Warmup and cool down become a continuous line. An abstract study of the work between experiments.",
    note: "Line study · Blue to teal, on a neutral field",
    banner: "editorial",
  },
] as const;

export type Direction = (typeof directions)[number]["key"];
