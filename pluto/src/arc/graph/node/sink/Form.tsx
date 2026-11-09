// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";

const path = deep.path<{ channel: channel.Key; value: number }>();

export const Form = (): ReactElement => (
  <Flex.Box x>
    <Base.Field path={path.channel}>
      {({ value, onChange }) => (
        <Channel.SelectSingle
          value={value}
          onChange={(key: channel.Key | null) => onChange(key ?? 0)}
          allowNone
        />
      )}
    </Base.Field>
    <Base.NumericField path={path.value} grow />
  </Flex.Box>
);
