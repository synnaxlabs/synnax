// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { type FC, type ReactNode } from "react";

import { type Theming } from "@/theming";

export interface FormProps {
  /** actions render in the right corner of the form's tab strip. */
  actions?: ReactNode;
  schematicKey?: string;
}

export type NodeProps<Config extends object = object> = {
  nodeKey: string;
  selected: boolean;
  onConfigChange: (data: Partial<Config>) => void;
  config: Config;
  position?: xy.XY;
  draggable?: boolean;
};

export type PreviewProps<P extends object = object> = P & {
  scale?: number;
};

export type Node<Config extends object = object> = FC<NodeProps<Config>>;

export interface Spec<Variant extends string = string, Config extends object = object> {
  key: Variant;
  name: string;
  Form: FC<FormProps>;
  Node: Node<Config>;
  defaultConfig: (t: Theming.Theme) => Config;
  Preview: FC<PreviewProps<Config>>;
  zIndex: number;
  needsPosition?: boolean;
}
