// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const HOW_TO_NAV: PageNavNode = {
  key: "how-to",
  name: "How-To Guides",
  children: [
    {
      key: "/reference/control/arc/how-to/unit-conversions",
      href: "/reference/control/arc/how-to/unit-conversions",
      name: "Unit Conversions",
    },
    {
      key: "/reference/control/arc/how-to/sensor-averaging",
      href: "/reference/control/arc/how-to/sensor-averaging",
      name: "Sensor Averaging",
    },
    {
      key: "/reference/control/arc/how-to/derived-calculations",
      href: "/reference/control/arc/how-to/derived-calculations",
      name: "Derived Calculations",
    },
    {
      key: "/reference/control/arc/how-to/rate-of-change",
      href: "/reference/control/arc/how-to/rate-of-change",
      name: "Rate of Change",
    },
    {
      key: "/reference/control/arc/how-to/sensor-validation",
      href: "/reference/control/arc/how-to/sensor-validation",
      name: "Sensor Validation",
    },
    {
      key: "/reference/control/arc/how-to/alarms",
      href: "/reference/control/arc/how-to/alarms",
      name: "Alarms",
    },
    {
      key: "/reference/control/arc/how-to/bang-bang-control",
      href: "/reference/control/arc/how-to/bang-bang-control",
      name: "Bang-Bang Control",
    },
    {
      key: "/reference/control/arc/how-to/test-sequences",
      href: "/reference/control/arc/how-to/test-sequences",
      name: "Test Sequences",
    },
  ],
};
