// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

const RATE_INPUT_PROPS = { endContent: "Hz" } as const;

/** Edits how often a read task queries polled messages and how long it waits. */
export const Poll = (): ReactElement => (
  <Flex.Box x>
    <Form.NumericField
      path="config.rate"
      label="Poll rate"
      inputProps={RATE_INPUT_PROPS}
    />
    <Form.Field<TimeSpan> path="config.timeout" label="Reply timeout">
      {({ value, onChange, ...rest }) => (
        <Input.Numeric
          {...rest}
          value={new TimeSpan(value).milliseconds}
          onChange={(ms) => onChange(TimeSpan.milliseconds(ms))}
          endContent="ms"
        />
      )}
    </Form.Field>
  </Flex.Box>
);
