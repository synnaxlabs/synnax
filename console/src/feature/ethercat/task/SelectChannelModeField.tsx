// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement, useCallback } from "react";

import {
  type ChannelMode,
  type ChannelSchemas,
  type ReadChannel,
  type WriteChannel,
} from "@/feature/ethercat/task/types";

const Base = Form.buildSelectField<ChannelMode>({
  fieldKey: "type",
  fieldProps: { label: "Mode", showHelpText: false },
  inputProps: {
    allowNone: false,
    resourceName: "channel mode",
    children: (
      <>
        <Select.Item itemKey="automatic">Automatic (PDO)</Select.Item>
        <Select.Item itemKey="manual">Manual (address)</Select.Item>
      </>
    ),
  },
});

export interface SelectChannelModeFieldProps {
  path: string;
  schemas: ChannelSchemas;
}

export const SelectChannelModeField = ({
  path,
  schemas,
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
      const next = schemas[value].parse({ type: value });
      set(parentPath, {
        ...next,
        key: prevParent.key,
        device: prevParent.device,
        name: prevParent.name,
        disabled: prevParent.disabled,
        type: value,
      });
    },
    [schemas],
  );
  return <Base path={path} onChange={handleChange} />;
};
