// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/MacroSelect.css";

import { Dialog, Flex, Input, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { getAllMacroDefinitions } from "@/perf/macros/registry";
import { type MacroType } from "@/perf/macros/types";

export interface MacroSelectProps {
  value: MacroType[];
  onChange: (value: MacroType[]) => void;
  disabled?: boolean;
}

export const MacroSelect = ({
  value,
  onChange,
  disabled,
}: MacroSelectProps): ReactElement => {
  const definitions = useMemo(() => getAllMacroDefinitions(), []);

  const handleToggle = useCallback(
    (type: MacroType) => {
      if (value.includes(type)) 
        onChange(value.filter((v) => v !== type));
       else 
        onChange([...value, type]);
      
    },
    [value, onChange],
  );

  const selectedCount = value.length;
  const label =
    selectedCount === 0
      ? "Select macros"
      : selectedCount === 1
        ? definitions.find((d) => d.type === value[0])?.name ?? "1 macro"
        : `${selectedCount} macros`;

  return (
    <Dialog.Frame variant="connected">
      <Dialog.Trigger variant="outlined" size="small" disabled={disabled}>
        {label}
      </Dialog.Trigger>
      <Dialog.Dialog bordered rounded className="console-perf-macro-select-dialog">
        <Flex.Box y>
          {definitions.map((def) => (
            <Flex.Box key={def.type} x align="center">
              <Input.Checkbox
                value={value.includes(def.type)}
                onChange={() => handleToggle(def.type)}
                disabled={disabled}
                size="tiny"
              />
              <Text.Text level="small">{def.name}</Text.Text>
            </Flex.Box>
          ))}
          {definitions.length === 0 && (
            <Text.Text level="small">No macros available</Text.Text>
          )}
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
