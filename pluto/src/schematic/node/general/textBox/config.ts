// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/x";

import { type Flex } from "@/flex";
import { type Label } from "@/schematic/node/common/label";
import { type LabeledConfig } from "@/schematic/node/common/symbol/factories";
import { type Text } from "@/text";

export interface Config extends LabeledConfig {
  color?: color.Crude;
  width?: number;
  align?: Flex.Alignment;
  autoFit?: boolean;
  level?: Text.Level;
  value?: string;
  label?: Label.Config;
}
