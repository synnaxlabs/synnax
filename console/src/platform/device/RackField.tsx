// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Rack } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface RackFieldProps {
  /** The integration the rack must run, such as "opc". */
  integration: string;
}

/** The form's "rack" field: the Driver that makes the connection. */
export const RackField = ({ integration }: RackFieldProps): ReactElement => (
  <Form.Field<rack.Key> path="rack" label="Connect from" required>
    {({ value, onChange }) => (
      <Rack.SelectSingle
        value={value}
        onChange={onChange}
        initialQuery={{ where: { integrations: { any: [integration] } } }}
      />
    )}
  </Form.Field>
);
