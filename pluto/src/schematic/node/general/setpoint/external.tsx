// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Label } from "@/schematic/node/common/label";
import { type Config, VARIANT } from "@/schematic/node/general/setpoint/config";
import { SetpointForm } from "@/schematic/node/general/setpoint/Form";
import { Setpoint } from "@/schematic/node/general/setpoint/Primitive";
import { Symbol } from "@/schematic/node/general/setpoint/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export * from "@/schematic/node/general/setpoint/config";

export const defaultConfig = (t: Theming.Theme): Config => ({
  variant: VARIANT,
  orientation: "left",
  units: "mV",
  color: t.colors.gray.l11,
  size: "small",
  label: Label.defaultConfig("Setpoint"),
  control: { show: true },
});

const Preview = ({ ...rest }: Config): ReactElement => (
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

export const spec: Spec<typeof VARIANT, Config> = {
  key: VARIANT,
  name: "Setpoint",
  Form: SetpointForm,
  Node: Symbol,
  Preview,
  defaultConfig,
  zIndex: 4,
};
