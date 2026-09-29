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

export interface NegatedSwitchFieldProps extends Omit<
  Form.FieldProps<boolean, boolean>,
  "children"
> {}

/**
 * Binds a switch to a stored boolean that states its non-default condition, such as
 * `fillHidden` or `dblClickNavDisabled`. The switch reads and writes the affirmative,
 * so `label` names what the switch turns on.
 */
export const NegatedSwitchField = (props: NegatedSwitchFieldProps): ReactElement => (
  <Form.Field<boolean> {...props}>
    {({ value, onChange, ...cp }) => (
      <Input.Switch {...cp} value={!value} onChange={(v) => onChange(!v)} />
    )}
  </Form.Field>
);
