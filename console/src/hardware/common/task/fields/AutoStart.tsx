// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";

export const AutoStart = () => (
  <Form.SwitchField
    label="Auto Start"
    path="config.autoStart"
    defaultValue={false}
    hideIfNull
  />
);
