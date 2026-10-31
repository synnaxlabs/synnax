// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Access, Ranger } from "@synnaxlabs/pluto";
import { type NumericTimeRange } from "@synnaxlabs/x";
import { memo } from "react";

import { CSS } from "@/platform/css";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Session } from "@/session";

export interface ItemProps extends List.ItemProps<ranger.Key> {
  showParent?: boolean;
  showLabels?: boolean;
  showTimeRange?: boolean;
  showFavorite?: boolean;
}

const Base = ({
  showParent = true,
  showLabels = true,
  showTimeRange = true,
  showFavorite = true,
  ...props
}: ItemProps) => {
  const { itemKey } = props;
  const openTab = Panel.useOpenTab();
  const item = List.useItem<ranger.Key, ranger.Range>(itemKey);
  const canEdit = Access.useUpdateGranted(ranger.ontologyID(itemKey));
  const { update: rename } = Session.Range.useRename();
  const initialValues =
    item == null
      ? undefined
      : {
          ...item.payload,
          labels: item.labels?.map((l) => l.key) ?? [],
          parent: item.parent?.key ?? "",
          timeRange: item.timeRange.numeric,
        };
  const { form } = Ranger.useForm({
    query: null,
    initialValues,
    sync: true,
    autoSave: true,
  });
  if (initialValues == null || item == null) return null;

  const { name, parent, labels, timeRange } = item;

  const handleActivate = () =>
    openTab({ variant: "resource", resource: ranger.ontologyID(itemKey) });

  return (
    <Select.Item<ranger.Key>
      className={CSS.BE("range", "list-item")}
      aria-label={name}
      checkable
      onActivate={handleActivate}
      justify="between"
      {...props}
    >
      <Form.Form<typeof Ranger.formSchema> {...form}>
        <Flex.Box x align="center" gap="tiny" grow>
          <Form.Field<NumericTimeRange>
            path="timeRange"
            showLabel={false}
            padHelpText={false}
          >
            {({ value, onChange }) => (
              <Ranger.StageButton
                value={value}
                onChange={onChange}
                iconOnly
                disabled={!canEdit}
              />
            )}
          </Form.Field>
          <Ranger.Breadcrumb
            name={name}
            parent={parent}
            showParent={showParent}
            overflow="nowrap"
            nameID={List.itemNameID(itemKey)}
            onRename={canEdit ? (name) => rename({ key: itemKey, name }) : undefined}
          />
        </Flex.Box>
        <Flex.Box x align="center" className={CSS.BE("range", "list-item-trailing")}>
          {showLabels && labels != null && labels.length > 0 && (
            <Tag.Tags variant="text">
              {labels.map(({ key, name, color }) => (
                <Tag.Tag key={key} color={color} size="small">
                  {name}
                </Tag.Tag>
              ))}
            </Tag.Tags>
          )}
          {showTimeRange && (
            <Telem.Text.TimeRange
              className={CSS.BE("range", "time-range")}
              level="tiny"
              role="group"
              aria-label="Time range"
            >
              {timeRange}
            </Telem.Text.TimeRange>
          )}
          {showFavorite && <Range.FavoriteButton range={item} reveal />}
        </Flex.Box>
      </Form.Form>
    </Select.Item>
  );
};

export const Item = memo(Base);
