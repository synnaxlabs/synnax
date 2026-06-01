// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { renderProp } from "@/component/renderProp";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Status } from "@/status";

const selectStatus = renderProp(Status.Select);
const selectVariant = renderProp(Status.SelectVariant);

const VARIANT_STYLE: CSSProperties = { width: "30rem" };

export const Form = (): ReactElement => (
  <Flex.Box y grow empty>
    <Base.Field<string> path="key_or_name" label="Status">
      {selectStatus}
    </Base.Field>
    <Flex.Box x grow>
      <Base.Field<status.Variant> path="variant" style={VARIANT_STYLE}>
        {selectVariant}
      </Base.Field>
      <Base.TextField path="message" grow />
    </Flex.Box>
  </Flex.Box>
);
