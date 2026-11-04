// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";

const path = deep.path<schematic.CylinderNodeConfig>();
export const CylinderForm = (): ReactElement => (
  <Base.Sections x>
    <Base.Section title="Label">
      <Label.Form path={path.label} />
    </Base.Section>
    <Base.Section title="Appearance">
      <Form.ColorField path={path.strokeColor} label="Stroke" />
      <Form.FillField />
    </Base.Section>
    <Base.Section title="Dimensions">
      <Base.NumericField
        path={path.dimensions.width}
        label="Width"
        inputProps={Form.DIMENSIONS_INPUT_PROPS}
      />
      <Base.NumericField
        path={path.dimensions.height}
        label="Height"
        inputProps={Form.DIMENSIONS_INPUT_PROPS}
      />
    </Base.Section>
    <Orientation.Section path={path} />
  </Base.Sections>
);
