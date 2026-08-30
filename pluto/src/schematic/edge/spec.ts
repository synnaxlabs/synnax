// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type FC } from "react";
import { type z } from "zod";

import { type FormProps } from "@/schematic/node/spec";
import { type Diagram } from "@/vis/diagram";

export interface EdgeProps<Config extends object = object> extends Diagram.EdgeProps {
  onChange: (p: Partial<schematic.EdgeConfig>) => void;
  config: Config;
}

export type Edge<Config extends object = object> = FC<EdgeProps<Config>>;

export interface Spec<
  Variant extends string = string,
  P extends { variant: Variant } = { variant: Variant },
> {
  key: Variant;
  name: string;
  configZ: z.ZodType<P>;
  Form: FC<FormProps>;
  Edge: Edge<P>;
  defaultConfig: () => P;
}
