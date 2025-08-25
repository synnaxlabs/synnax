// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Tooltip } from "@synnaxlabs/pluto";

export const DataSaving = () => (
  <Tooltip.Dialog location={{ x: "center", y: "bottom" }}>
    Save task data on Synnax core
    <Form.SwitchField label="Data Saving" path="config.dataSaving" />
  </Tooltip.Dialog>
);
