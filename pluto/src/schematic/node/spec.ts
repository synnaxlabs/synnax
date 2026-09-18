// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type xy } from "@synnaxlabs/x";
import { type FC, type ReactNode } from "react";

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

/**
 * Spec describes how to render and edit one node variant. The default parameters give
 * the erased view that holds a spec of any variant.
 */
export interface Spec<
  Variant extends string = schematic.NodeConfigType,
  Config extends object = schematic.NodeConfig,
> {
  key: Variant;
  name: string;
  /** The text written into the symbol's label on creation. Defaults to name. */
  label?: string;
  Form: FC<FormProps>;
  Node: Node<Config>;
  Preview: FC<PreviewProps<Config>>;
  zIndex: number;
  needsPosition?: boolean;
}
