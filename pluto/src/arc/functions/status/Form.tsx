// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Form as Core } from "@/form";
import { Status } from "@/status";

export const Form = (): ReactElement => (
  <Flex.Box y grow empty>
    <Core.Field<string> path="statusKey" label="Status">
      {(p) => <Status.Select {...p} />}
    </Core.Field>
    <Flex.Box x grow>
      <Core.Field<status.Variant> path="variant" style={{ width: "30rem" }}>
        {(p) => <Status.SelectVariant {...p} />}
      </Core.Field>
      <Core.TextField path="message" grow />
    </Flex.Box>
  </Flex.Box>
);
