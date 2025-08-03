// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { List } from "@/list";
import { Ranger } from "@/ranger";
import { Breadcrumb } from "@/ranger/Breadcrumb";
import { TimeRangeChip, type TimeRangeChipProps } from "@/ranger/TimeRangeChip";
import { Select } from "@/select";
import { Tag } from "@/tag";

interface ListItemProps
  extends List.ItemProps<ranger.Key>,
    Pick<TimeRangeChipProps, "showAgo" | "showSpan"> {
  showParent?: boolean;
  showLabels?: boolean;
  onStar?: (starred: boolean) => void;
  onStageChange?: (stage: ranger.Stage) => void;
}

export const ListItem = ({
  itemKey,
  showParent = true,
  showLabels = true,
  onStar,
  showAgo,
  showSpan,
  onStageChange,
  ...rest
}: ListItemProps): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  if (item == null) return null;
  const { name, timeRange, parent, labels, stage } = item;
  return (
    <Select.ListItem
      className={CSS(CSS.BE("range", "list-item"))}
      itemKey={itemKey}
      justify="between"
      {...rest}
    >
      <Flex.Box x align="center" empty>
        <Ranger.SelectStage
          value={stage}
          allowNone={false}
          onChange={(v: ranger.Stage | null) => v != null && onStageChange?.(v)}
          onClick={(e) => e.stopPropagation()}
          variant="floating"
          location="bottom"
          triggerProps={{ variant: "text" }}
        />
        <Breadcrumb name={name} parent={parent} showParent={showParent} />
      </Flex.Box>
      <Flex.Box x>
        {showLabels && (
          <Tag.Tags>
            {labels?.map(({ key, name, color }) => (
              <Tag.Tag key={key} color={color} size="small">
                {name}
              </Tag.Tag>
            ))}
          </Tag.Tags>
        )}
        <TimeRangeChip
          level="small"
          timeRange={timeRange}
          showAgo={showAgo}
          showSpan={showSpan}
        />
      </Flex.Box>
    </Select.ListItem>
  );
};
