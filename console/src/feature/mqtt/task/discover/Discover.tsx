// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/discover/Discover.css";

import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { math } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { useState } from "react";

import type { Device } from "@/feature/mqtt/device/types";
import { TagPicker, type TagPicks } from "@/feature/mqtt/task/discover/Tags";
import { TopicPicker, type TopicPicks } from "@/feature/mqtt/task/discover/Topics";
import type { HeldEntry } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";
import { Modals } from "@/platform/modals";

type Mode = "topics" | "sparkplug";

/** The values picked from topics and the tags picked from edge nodes. */
export interface Picks {
  topics: TopicPicks;
  tags: TagPicks;
}

/** What a dialog adds to a task: the entries of a read or the targets of a write. */
export interface Kind<E extends HeldEntry> {
  /** Names the dialog. */
  title: string;
  /** Says what a pick becomes. It shows while nothing is picked. */
  hint: string;
  /** Names what one picked value of a topic adds, such as "field". */
  noun: string;
  /** Names what one picked tag adds. Defaults to noun. */
  tagNoun?: string;
  /** Each topic takes one value, and a topic that the task holds takes none. */
  single?: boolean;
  /** @returns what the picks add to a task that holds the entries. */
  create: (picks: Picks, entries: E[]) => E[];
  /** @returns what a topic added by name adds. */
  createTopic: (topic: string) => E;
  /** @returns what a tag added by name adds. Absent, the dialog lists no tags. */
  createTag?: () => E;
}

interface Params<E extends HeldEntry> {
  device: Device;
  /** What the task holds now. */
  entries: E[];
}

type DiscoverProps<E extends HeldEntry> = Modals.ContentProps<Params<E>, E[]> & {
  kind: Kind<E>;
};

const Discover = <E extends HeldEntry>({
  device,
  entries,
  close,
  kind,
}: DiscoverProps<E>) => {
  const { createTag, noun, tagNoun = noun } = kind;
  const [mode, setMode] = useState<Mode>("topics");
  const [topicPicks, setTopicPicks] = useState<TopicPicks>(() => new Map());
  const [tagPicks, setTagPicks] = useState<TagPicks>(() => new Map());

  const picked = (): E[] =>
    kind.create({ topics: topicPicks, tags: tagPicks }, entries);

  const values = math.sum([...topicPicks.values()].map((p) => p.pointers.size));
  const count = values + tagPicks.size;
  const parts: string[] = [];
  if (values > 0) {
    const topics = pluralize("topic", topicPicks.size, true);
    parts.push(`${pluralize("value", values, true)} from ${topics}`);
  }
  if (tagPicks.size > 0) parts.push(pluralize("Sparkplug B tag", tagPicks.size, true));
  const summary = parts.length === 0 ? kind.hint : `${parts.join(" and ")} selected`;

  return (
    <ICD.Picker.Root value={mode} className={CSS.B("mqtt-discover")}>
      <Modal.Header icon={<Icon.Logo.MQTT />}>{[kind.title, device.name]}</Modal.Header>
      <ICD.Picker.Bar>
        {createTag != null && (
          <Select.Buttons<Mode> value={mode} onChange={setMode}>
            <Select.Item<Mode> itemKey="topics">JSON topics</Select.Item>
            <Select.Item<Mode> itemKey="sparkplug">Sparkplug B</Select.Item>
          </Select.Buttons>
        )}
      </ICD.Picker.Bar>
      <ICD.Picker.Frame itemKey="topics">
        <TopicPicker
          device={device}
          entries={entries}
          single={kind.single}
          value={topicPicks}
          onChange={setTopicPicks}
          onAddByName={(topic) => close([...picked(), kind.createTopic(topic)])}
        />
      </ICD.Picker.Frame>
      {createTag != null && (
        <ICD.Picker.Frame itemKey="sparkplug">
          <TagPicker
            device={device}
            entries={entries}
            value={tagPicks}
            onChange={setTagPicks}
            onAddByName={() => close([...picked(), createTag()])}
          />
        </ICD.Picker.Frame>
      )}
      <ICD.Picker.Footer
        counts={[
          [noun, values],
          [tagNoun, tagPicks.size],
        ]}
        idle={mode === "sparkplug" ? tagNoun : noun}
        onAdd={() => close(picked())}
        onCancel={() => close()}
      >
        <Text.Text level="small" color={count > 0 ? 10 : 9}>
          {summary}
        </Text.Text>
      </ICD.Picker.Footer>
    </ICD.Picker.Root>
  );
};

/**
 * Creates a hook that opens a dialog listing what the broker publishes, so the user can
 * pick values. The dialog resolves with what the kind creates from the picks, or null
 * when the user closes it.
 */
export const createDiscoverModal = <E extends HeldEntry>(
  kind: Kind<E>,
): Modals.PromptHook<Params<E>, E[]> => {
  const Content = (props: Modals.ContentProps<Params<E>, E[]>) => (
    <Discover {...props} kind={kind} />
  );
  return Modals.createPrompt<E[], Params<E>>(Content);
};
