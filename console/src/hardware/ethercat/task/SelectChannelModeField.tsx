// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { deep, type record } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import {
  AUTOMATIC_TYPE,
  type ChannelMode,
  MANUAL_TYPE,
  type ReadChannel,
  type WriteChannel,
} from "@/hardware/ethercat/task/types";

interface ChannelModeEntry extends record.KeyedNamed<ChannelMode> {}

const DATA: ChannelModeEntry[] = [
  { key: AUTOMATIC_TYPE, name: "Automatic (PDO)" },
  { key: MANUAL_TYPE, name: "Manual (Address)" },
];

const Base = Form.buildSelectField<ChannelMode, ChannelModeEntry>({
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

export interface SelectChannelModeFieldProps {
  path: string;
  zeroChannels: Record<ChannelMode, ReadChannel | WriteChannel>;
}

export const SelectChannelModeField = ({
  path,
  zeroChannels,
}: SelectChannelModeFieldProps): ReactElement => {
  const handleChange = useCallback(
    (
      value: ChannelMode,
      { get, set, path: fieldPath }: Form.ContextValue & { path: string },
    ) => {
      const prevType = get(fieldPath).value;
      if (prevType === value) return;
      const parentPath = fieldPath.slice(0, fieldPath.lastIndexOf("."));
      const prevParent = get<ReadChannel | WriteChannel>(parentPath).value;
      const next = deep.copy(zeroChannels[value]);
      set(parentPath, {
        ...next,
        key: prevParent.key,
        device: prevParent.device,
        name: prevParent.name,
        enabled: prevParent.enabled,
        type: value,
      });
    },
    [zeroChannels],
  );
  return <Base path={path} onChange={handleChange} />;
};
