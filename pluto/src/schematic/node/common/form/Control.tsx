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
import { type ReactElement } from "react";

import { Control } from "@/schematic/node/common/control";

// A symbol with no control state shows none, so it reads as a hidden one.
const HIDDEN = Control.stateConfigZ.parse({ hidden: true });

/**
 * Shows or hides a symbol's control state. A symbol with no control state reads as
 * off, and turning it on adds the default state.
 */
export const ControlChipField = (): ReactElement => (
  <Form.Field<Control.StateConfig>
    path="control"
    label="Control chip"
    defaultValue={HIDDEN}
    padHelpText={false}
  >
    {({ value, onChange, ...rest }) => (
      <Input.Switch
        {...rest}
        value={!value.hidden}
        onChange={(shown) => onChange({ ...value, hidden: !shown })}
      />
    )}
  </Form.Field>
);
