// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/label/filter/property.css";

import { type label } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Flux, Label } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { View } from "@/platform/view";

/** How a condition reads the labels of a resource. `empty` matches no labels. */
export type Match = "any" | "all" | "none" | "empty";

/** A condition on the labels of a resource. */
export interface Condition {
  match: Match;
  keys: label.Key[];
}

/** The part of a filter that the labels property reads and writes. */
export interface Where {
  and?: Where[];
  not?: Where;
  labels?: label.Key[];
  labeled?: boolean;
}

const soleLabel = (node: Where): label.Key | null =>
  View.Filter.soleField(node) === "labels" && node.labels?.length === 1
    ? node.labels[0]
    : null;

const decode = (node: Where): Condition | null => {
  const field = View.Filter.soleField(node);
  if (field === "labels" && node.labels != null)
    return { match: "any", keys: node.labels };
  if (field === "labeled" && node.labeled === false)
    return { match: "empty", keys: [] };
  if (
    field === "not" &&
    node.not?.labels != null &&
    View.Filter.soleField(node.not) === "labels"
  )
    return { match: "none", keys: node.not.labels };
  if (field === "and" && node.and != null && node.and.length > 0) {
    const keys = node.and.map(soleLabel);
    if (keys.every((key) => key != null)) return { match: "all", keys };
  }
  return null;
};

const encode = ({ match, keys }: Condition): Where => {
  switch (match) {
    case "any":
      return { labels: keys };
    case "all":
      return { and: keys.map((key) => ({ labels: [key] })) };
    case "none":
      return { not: { labels: keys } };
    case "empty":
      return { labeled: false };
  }
};

type KeyMatch = Exclude<Match, "empty">;

const MATCH_NAMES: Record<KeyMatch, string> = {
  any: "has any of",
  all: "has all of",
  none: "has none of",
};

const KEY_MATCHES = Object.keys(MATCH_NAMES) as KeyMatch[];

/** The option for resources with no labels. Only the picker holds this key. */
const NO_LABELS = "no-labels";

const labelItem = Component.renderProp(Label.ListItem);

/** @returns one or two names, or else a count. */
const summarize = (names: string[]): string =>
  names.length > 0 && names.length <= 2 ? names.join(", ") : `${names.length} labels`;

const createPicker = (unlabeled: boolean) => {
  const Picker = ({
    value,
    onChange,
  }: View.Filter.PickerProps<Condition>): ReactElement => {
    const { data, getItem, subscribe, status, more, search } = Flux.useList(
      (client) => client.labels,
      { name: "labels" },
    );
    let keys: label.Key[] = [];
    if (value != null) keys = value.match === "empty" ? [NO_LABELS] : value.keys;
    const handleChange = (next: label.Key[]): void => {
      if (next.includes(NO_LABELS) && !keys.includes(NO_LABELS))
        return onChange({ match: "empty", keys: [] });
      const picked = next.filter((key) => key !== NO_LABELS);
      if (picked.length === 0) return onChange(null);
      const match = value == null || value.match === "empty" ? "any" : value.match;
      onChange({ match, keys: picked });
    };
    return (
      <Select.Frame<label.Key, label.Label | undefined>
        multiple
        value={keys}
        onChange={handleChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
      >
        <Select.Dialog surface="menu" aria-label="Labels">
          <Select.Body<label.Key>
            resourceName="label"
            onSearch={search}
            status={status}
            fixedItems={
              unlabeled ? (
                <Select.Item<label.Key> itemKey={NO_LABELS} textValue="No labels">
                  <Icon.Label />
                  No labels
                </Select.Item>
              ) : undefined
            }
          >
            {labelItem}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    );
  };
  return Picker;
};

const Operator = ({
  value,
  onChange,
}: View.Filter.ConditionProps<Condition>): ReactElement => {
  // A filter can keep the key of a deleted label, so count the labels that resolve.
  const { data: labels = [] } = Label.useResultMultiple({ keys: value.keys });
  const { match } = value;
  if (match === "empty") return <View.Filter.ChipOperator>is</View.Filter.ChipOperator>;
  return (
    <Dialog.Frame variant="floating" location={View.Filter.BELOW}>
      <Select.Frame<KeyMatch, undefined>
        value={match}
        onChange={(next: KeyMatch) => onChange({ ...value, match: next })}
        closeDialogOnSelect
      >
        <View.Filter.ChipTrigger textColor={9}>
          {match === "any" && labels.length === 1 ? "has" : MATCH_NAMES[match]}
        </View.Filter.ChipTrigger>
        <Select.Dialog surface="menu" aria-label="Match">
          <Select.List>
            {KEY_MATCHES.map((key) => (
              <Select.Item<KeyMatch> key={key} itemKey={key}>
                {MATCH_NAMES[key]}
              </Select.Item>
            ))}
          </Select.List>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

const Value = ({ value }: View.Filter.ConditionProps<Condition>): ReactElement => {
  const { data: labels = [] } = Label.useResultMultiple({ keys: value.keys });
  if (value.match === "empty") return <>No labels</>;
  return (
    <>
      <Flex.Box x empty className={CSS.BE("label-filter", "dots")}>
        {labels.map(({ key, color }) => (
          <Icon.Circle key={key} color={color} />
        ))}
      </Flex.Box>
      {summarize(labels.map(({ name }) => name))}
    </>
  );
};

export interface CreatePropertyArgs {
  /** Whether the picker offers "No labels", for a filter that reads `labeled`. */
  unlabeled?: boolean;
}

/** @returns the labels property of a view filter. */
export const createProperty = ({
  unlabeled = false,
}: CreatePropertyArgs = {}): View.Filter.Property<Where, Condition> => ({
  name: "Labels",
  icon: Icon.Label,
  decode,
  encode,
  Picker: createPicker(unlabeled),
  Operator,
  Value,
});
