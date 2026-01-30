// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import {
  AUTOMATIC_TYPE,
  type ChannelMode,
  MANUAL_TYPE,
} from "@/hardware/ethercat/task/types";

export interface ChannelModeEntry extends record.KeyedNamed<ChannelMode> {}

const DATA: ChannelModeEntry[] = [
  { key: AUTOMATIC_TYPE, name: "Automatic (PDO)" },
  { key: MANUAL_TYPE, name: "Manual (Address)" },
];

export interface SelectChannelModeFieldProps extends Omit<
  Form.SelectFieldProps<ChannelMode, ChannelModeEntry>,
  "data" | "entryRenderKey" | "columns"
> {}

export const SelectChannelModeField = Form.buildSelectField<
  ChannelMode,
  ChannelModeEntry
>({
  fieldKey: "type",
  fieldProps: {
    label: "Mode",
    showHelpText: false,
  },
  inputProps: {
    allowNone: false,
    resourceName: "channel mode",
    data: DATA,
  },
});
