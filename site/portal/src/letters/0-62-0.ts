// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Letter } from "@/server/letter";

export const LETTER: Letter = {
  version: "0.62.0",
  subject: "Synnax v0.62: ranges built around your test, Sift, and 10 kHz EtherCAT",
  title: "Ranges built around your test, Sift uploads, and EtherCAT at 10 kHz",
  hero: "releases/0-62-0/hero",
  opening: [
    "Synnax v0.62 is out. Here are the three changes that I am most excited about.",
    "EtherCAT also reads at 10 kHz on a USB network adapter, Arc stops a program " +
      "that runs too long, and one Settings tab holds your users and secrets.",
  ],
  features: [
    {
      area: "Ranges",
      title: "Ranges built around your test",
      text:
        "Start a test from the ranges toolbar and step through its phases with Next. " +
        "Every plot follows the run as it happens, with its T+ time above the plot.",
      href: "/releases#ranges-built-around-your-test",
      image: "releases/0-62-0/ranges",
    },
    {
      area: "Sift",
      title: "Send data to Sift",
      text:
        "Right-click a range and select Upload to Sift, or stream live data with a " +
        "Sift stream task. The Core sends the data, with no Driver to set up.",
      href: "/reference/ranges/sift",
      image: "releases/0-62-0/sift",
    },
    {
      area: "Line plots",
      title: "Named colors",
      text:
        "Link a line to a named color from the Library tab. Change the color one " +
        "time, and every plot that uses it changes with it.",
      href: "/releases#named-colors",
      image: "releases/0-62-0/colors",
    },
  ],
  breaking:
    "Some client calls, Arc unit rules, and Console features changed. Read the " +
    "breaking changes in the release notes before you upgrade.",
  closing: [
    "Reply to this email with questions or feedback. Your reply comes straight to me.",
    "Emiliano",
  ],
};
