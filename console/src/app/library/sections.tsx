// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { LABELS_SECTION } from "@/app/library/tab";
import {
  useColorsVisible,
  useEnumerationsVisible,
  useLabelsVisible,
} from "@/app/library/visible";
import { Color } from "@/feature/color";
import { Enumeration } from "@/feature/enumeration";
import { Label } from "@/feature/label";
import type { Settings } from "@/platform/settings";

export const GROUPS: Settings.Group[] = [
  {
    key: "core",
    name: "Core",
    sections: [
      {
        key: LABELS_SECTION,
        name: "Labels",
        icon: <Icon.Label />,
        Content: Label.Settings,
        useVisible: useLabelsVisible,
      },
      {
        key: "colors",
        name: "Colors",
        icon: <Icon.Swatches />,
        Content: Color.Settings,
        useVisible: useColorsVisible,
      },
      {
        key: "enumerations",
        name: "Enumerations",
        icon: <Icon.Enumeration />,
        Content: Enumeration.Settings,
        useVisible: useEnumerationsVisible,
      },
    ],
  },
];
