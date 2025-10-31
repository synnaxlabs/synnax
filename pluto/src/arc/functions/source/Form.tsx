// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form as Core } from "@/form";

export const Form = (): ReactElement => (
  <Core.Field<channel.Key> path="channel">
    {({ value, onChange }) => (
      <Channel.SelectSingle value={value} onChange={onChange} />
    )}
  </Core.Field>
);
