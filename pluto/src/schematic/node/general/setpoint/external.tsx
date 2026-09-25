// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { SetpointForm } from "@/schematic/node/general/setpoint/Form";
import { Setpoint } from "@/schematic/node/general/setpoint/Primitive";
import { Symbol } from "@/schematic/node/general/setpoint/Symbol";
import { type Spec } from "@/schematic/node/spec";

const Preview = ({
  label: _label,
  ...rest
}: schematic.SetpointNodeConfig): ReactElement => (
  <Setpoint
    {...rest}
    onChange={() => {}}
    className={CSS.BM("setpoint", "preview")}
    disabled
  />
);

export const spec: Spec<"setpoint", schematic.SetpointNodeConfig> = {
  key: "setpoint",
  name: "Setpoint",
  Form: SetpointForm,
  Node: Symbol,
  Preview,
  zIndex: 4,
};
