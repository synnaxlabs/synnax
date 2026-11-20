// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { deep } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Border } from "@/schematic/node/common/border";
import { STROKE_WIDTH_INPUT_PROPS } from "@/schematic/node/common/form/input";

const path = deep.path<{ strokeWidth?: number }>();

/** StrokeWidthField edits a border width and shows the default while it is absent. */
export const StrokeWidthField = (): ReactElement => (
  <Form.Field path={path.strokeWidth} label="Border width">
    {({ value, onChange }) => (
      <Input.Numeric
        value={value ?? Border.DEFAULT_STROKE_WIDTH}
        onChange={onChange}
        {...STROKE_WIDTH_INPUT_PROPS}
      />
    )}
  </Form.Field>
);
