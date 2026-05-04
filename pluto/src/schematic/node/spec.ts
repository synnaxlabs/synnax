// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";

import { type SymbolFormProps as NodeFormProps } from "@/schematic/node/common/forms";
import {
  type NodeProps,
  type PreviewProps,
} from "@/schematic/node/common/symbol/factories";
import { type Theming } from "@/theming";

export interface Spec<Variant extends string = string, Config extends object = object> {
  key: Variant;
  name: string;
  Form: FC<NodeFormProps>;
  Node: FC<NodeProps<Config>>;
  defaultConfig: (t: Theming.Theme) => Config;
  Preview: FC<PreviewProps<Config>>;
  zIndex: number;
  needsPosition?: boolean;
}
