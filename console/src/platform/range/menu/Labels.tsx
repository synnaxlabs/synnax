// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Access, Flux, Label as PLabel } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Label } from "@/platform/label";

interface ItemProps extends List.ItemRenderProps<label.Key> {
  /** The labels that only some of the ranges hold. */
  partial: Set<label.Key>;
}

const Item = ({ itemKey, partial, ...rest }: ItemProps): ReactElement | null => {
  const item = List.useItem<label.Key, label.Label>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon.Circle color={item.color} size="2.5em" />
      {item.name}
      {partial.has(itemKey) && (
        <Icon.Subtract className={CSS.BE("range-menu", "partial")} aria-label="Some" />
      )}
    </Select.Item>
  );
};

export interface LabelsProps {
  ranges: ranger.Range[];
}

/**
 * Picks the labels of the ranges. A label that only some of them hold adds to all of
 * them on the first click and leaves all of them on the next.
 */
export const Labels = ({ ranges }: LabelsProps): ReactElement => {
  const { update } = PLabel.useEditFor();
  const canCreate = Access.useCreateGranted(label.TYPE_RESOURCE_ID);
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (c) => c.labels,
    { name: "labels" },
  );
  const held = (key: label.Key) => ranges.filter((r) => r.labels.includes(key));
  const all = new Set(ranges.flatMap((r) => r.labels));
  const common = [...all].filter((key) => held(key).length === ranges.length);
  const partial = new Set([...all].filter((key) => !common.includes(key)));
  const add = (key: label.Key) =>
    ranges
      .filter((r) => !r.labels.includes(key))
      .forEach((r) =>
        update({ id: ranger.resourceID(r.key), added: [key], removed: [] }),
      );
  const remove = (key: label.Key) =>
    held(key).forEach((r) =>
      update({ id: ranger.resourceID(r.key), added: [], removed: [key] }),
    );
  const handleChange = (_: label.Key[], { clicked }: { clicked: label.Key | null }) => {
    if (clicked == null) return;
    if (common.includes(clicked)) remove(clicked);
    else add(clicked);
  };
  return (
    <Menu.Sub itemKey="labels" textValue="Labels">
      <Menu.SubTrigger>
        <Icon.Label />
        Labels
      </Menu.SubTrigger>
      <Select.Frame<label.Key, label.Label | undefined>
        multiple
        selectedFirst
        value={common}
        onChange={handleChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
      >
        <Select.Dialog surface="menu" filtered aria-label="Labels">
          <Select.Body<label.Key>
            resourceName="label"
            onSearch={search}
            status={status}
            footer={canCreate && <Label.CreateFooter onCreate={add} />}
          >
            {({ key, ...p }) => <Item key={key} {...p} partial={partial} />}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    </Menu.Sub>
  );
};
