// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import { Form as Core } from "@/form";
import { Input } from "@/input";

export const Form = () => (
  <Core.Field<number> path="duration">
    {({ value, onChange }) => (
      <Input.Numeric
        value={new TimeSpan(value).seconds}
        onChange={(v) => onChange(TimeSpan.seconds(v).nanoseconds)}
        endContent="s"
      />
    )}
  </Core.Field>
);
