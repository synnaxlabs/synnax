// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { SIZE_TEXT_LEVELS } from "@/component/text";
import { CSS } from "@/css";
import { useContext as useFrameContext } from "@/tabs/Frame";
import { useSelectorContext } from "@/tabs/Selector";
import { useTabContext } from "@/tabs/Tab";
import { Text } from "@/text";

export interface NameProps extends Omit<Text.EditableProps, "onChange"> {}

/**
 * Name renders a tab's name inside a Tab. When the enclosing Frame has an onRename
 * handler, the name becomes editable: double-clicking enters edit mode and committing
 * calls onRename with the tab's key and the new name. When the Frame has no onRename
 * handler, the name renders as static text.
 */
export const Name = ({ value, level, ...rest }: NameProps): ReactElement => {
  const { onRename } = useFrameContext("Tabs.Name");
  const { itemKey } = useTabContext("Tabs.Name");
  const { size } = useSelectorContext("Tabs.Name");
  const handleRename = useCallback(
    (name: string) => onRename?.(itemKey, name),
    [itemKey, onRename],
  );
  level ??= SIZE_TEXT_LEVELS[size];
  if (onRename == null)
    return (
      <Text.Text level={level} overflow="ellipsis" {...rest}>
        {value}
      </Text.Text>
    );
  return (
    <Text.Editable
      id={CSS.B(`tab-${itemKey}`)}
      level={level}
      value={value}
      onChange={handleRename}
      overflow="ellipsis"
      {...rest}
    />
  );
};
