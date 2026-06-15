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
import { Label } from "@/schematic/node/common/label";
import { SetpointForm } from "@/schematic/node/general/setpoint/Form";
import { Setpoint } from "@/schematic/node/general/setpoint/Primitive";
import { Symbol } from "@/schematic/node/general/setpoint/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigSetpoint => ({
  variant: "setpoint",
  orientation: "left",
  units: "mV",
  color: t.colors.gray.l11,
  size: "small",
  label: Label.defaultConfig("Setpoint"),
  control: { show: true },
});

const Preview = ({ ...rest }: schematic.NodeConfigSetpoint): ReactElement => (
  <Setpoint
    value={12}
    onChange={() => {}}
    units="mV"
    style={{ width: 120, transform: "scale(0.95)" }}
    className={CSS.BM("setpoint", "preview")}
    disabled
    {...rest}
  />
);

export const spec: Spec<"setpoint", schematic.NodeConfigSetpoint> = {
  key: "setpoint",
  name: "Setpoint",
  Form: SetpointForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
