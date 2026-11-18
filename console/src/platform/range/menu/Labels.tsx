// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, label, ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Access, Flux, Label, Synnax } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";

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

const CREATE = "create";

export interface LabelsProps {
  ranges: ranger.Range[];
}

/**
 * Picks the labels of the ranges. A label that only some of them hold adds to all of
 * them on the first click and leaves all of them on the next. Typed text that matches
 * no label creates one.
 */
export const Labels = ({ ranges }: LabelsProps): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const { update } = Label.useEditFor();
  const canCreate = Access.useCreateGranted(label.TYPE_RESOURCE_ID);
  const { colors } = Theming.use();
  const [term, setTerm] = useState("");
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
  const name = term.trim();
  const exists = data.some(
    (key) => getItem(key)?.name.toLowerCase() === name.toLowerCase(),
  );
  const handleCreate = () =>
    handleError(
      async () => {
        if (client == null) throw new DisconnectedError();
        // A new label takes the first palette color that no label uses yet.
        const used = new Set(
          data.flatMap((key) => {
            const c = getItem(key)?.color;
            return c == null ? [] : [color.hex(c)];
          }),
        );
        const palette = colors.visualization.palettes.default;
        const next = palette.find((c) => !used.has(color.hex(c))) ?? palette[0];
        const created = await client.labels.create({ name, color: next });
        add(created.key);
      },
      { message: "Failed to create label", name },
    );
  const handleChange = (_: label.Key[], { clicked }: { clicked: label.Key | null }) => {
    if (clicked == null) return;
    if (clicked === CREATE) return handleCreate();
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
        <Select.Dialog surface="menu" aria-label="Labels">
          <Select.Body<label.Key>
            resourceName="label"
            onSearch={(t) => {
              setTerm(t);
              search(t);
            }}
            status={status}
            footer={
              canCreate && name !== "" && !exists ? (
                <Select.Item<label.Key> itemKey={CREATE} textValue={name}>
                  <Icon.Add />
                  Create label "{name}"
                </Select.Item>
              ) : undefined
            }
          >
            {({ key, ...p }) => <Item key={key} {...p} partial={partial} />}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    </Menu.Sub>
  );
};
