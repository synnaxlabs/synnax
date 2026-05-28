// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Form } from "@/form";

export const ControlChipField = (
  props: Partial<Form.SwitchFieldProps>,
): ReactElement => (
  <Form.SwitchField
    path="control.show"
    label="Show control chip"
    hideIfNull
    optional
    padHelpText={false}
    {...props}
  />
);
