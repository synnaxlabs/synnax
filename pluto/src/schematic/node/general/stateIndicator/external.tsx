// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { NAME, StateIndicatorForm } from "@/schematic/node/general/stateIndicator/Form";
import { Primitive } from "@/schematic/node/general/stateIndicator/primitive";
import { Symbol } from "@/schematic/node/general/stateIndicator/Symbol";
import type { Spec } from "@/schematic/node/spec";

const Preview = ({
  strokeColor,
  size,
}: schematic.StateIndicatorNodeConfig): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <Primitive.StateIndicator
      matchedOptionKey="1"
      options={[{ key: "1", name: "Active", value: 1 }]}
      strokeColor={resolve(strokeColor)}
      size={size}
    />
  );
};

export const spec: Spec<"state_indicator", schematic.StateIndicatorNodeConfig> = {
  key: "state_indicator",
  name: NAME,
  Form: StateIndicatorForm,
  Node: Symbol,
  Preview,
  zIndex: 4,
};
