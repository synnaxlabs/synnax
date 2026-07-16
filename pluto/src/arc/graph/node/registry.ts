// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import z from "zod/v4";

import { Constant } from "@/arc/graph/node/constant";
import { Operator } from "@/arc/graph/node/operator";
import { Select } from "@/arc/graph/node/select";
import { Sink } from "@/arc/graph/node/sink";
import { Source } from "@/arc/graph/node/source";
import { StableFor } from "@/arc/graph/node/stable";
import { Status } from "@/arc/graph/node/status";
import { type Spec } from "@/arc/graph/node/types/spec";
import { Icon } from "@/icon";

// REGISTRY maps each Arc function type to its Spec. The key is the node config's "type"
// discriminant, matched against the Arc compiler's function templates.
export const REGISTRY = {
  ...Source.REGISTRY,
  ...Sink.REGISTRY,
  ...Constant.REGISTRY,
  ...Select.REGISTRY,
  ...Status.REGISTRY,
  ...Operator.REGISTRY,
  ...StableFor.REGISTRY,
} as const;

// Type is the union of registered function type discriminants.
export type Type = keyof typeof REGISTRY;

// configZ validates a node config against the registered function types, discriminating
// on "type". Config is the resulting union; ConfigOf narrows it to a single type.
export const configZ = z.discriminatedUnion("type", [
  Source.configZ,
  Sink.configZ,
  Constant.configZ,
  Select.configZ,
  Status.configZ,
  ...Operator.configZ.options,
  StableFor.configZ,
]);
export type Config = z.infer<typeof configZ>;
export type ConfigOf<T extends Type> = Extract<Config, { type: T }>;

// resolveSpec returns the Spec for the given function type. It throws NotFoundError if
// the type is not registered.
export const resolveSpec = (type: string): Spec<Type, Config> => {
  const spec = (REGISTRY as Record<string, Spec>)[type];
  if (spec == null) throw new NotFoundError(`Arc function ${type} not found`);
  return spec as Spec<Type, Config>;
};

export interface Group {
  key: string;
  name: string;
  Icon: Icon.FC;
  symbols: Type[];
}

export const GROUPS: Group[] = [
  {
    key: "basic",
    name: "Basic",
    Icon: Icon.Schematic,
    symbols: [...keysOf(Constant.REGISTRY), ...keysOf(Status.REGISTRY)],
  },
  {
    key: "telem",
    name: "Telemetry",
    Icon: Icon.Channel,
    symbols: [...keysOf(Source.REGISTRY), ...keysOf(Sink.REGISTRY)],
  },
  {
    key: "operator",
    name: "Operators",
    Icon: Icon.Add,
    symbols: keysOf(Operator.REGISTRY),
  },
  {
    key: "flow_control",
    name: "Flow Control",
    Icon: Icon.Select,
    symbols: [...keysOf(Select.REGISTRY), ...keysOf(StableFor.REGISTRY)],
  },
];

function keysOf<T extends string>(registry: Record<T, unknown>): Type[] {
  return Object.keys(registry) as Type[];
}
