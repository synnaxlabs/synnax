// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/discover/Discover.css";

import type { icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { math } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { useState } from "react";

import type { Device } from "@/feature/mqtt/device";
import { TagPicker, type TagPicks } from "@/feature/mqtt/task/discover/Tags";
import {
  countOf,
  TopicPicker,
  type TopicPicks,
} from "@/feature/mqtt/task/discover/Topics";
import type { HeldEntry } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";
import type { Modals } from "@/platform/modals";

type Mode = "topics" | "sparkplug";

const FORMAT_REQUIRED = "Select a message format to add topics.";

/** A message of an ICD. */
export interface Format {
  icd: icd.Key;
  message: icd.Message;
}

/** The values picked from topics and the tags picked from edge nodes. */
export interface Picks {
  topics: TopicPicks;
  tags: TagPicks;
  /** The message that each entry from a topic sends, when the kind picks one. */
  format: Format | null;
}

/** What a dialog adds to a task: the entries of a read or the targets of a write. */
export interface Kind<E> {
  /** Names the dialog. */
  title: string;
  /** Says what a picked value of a topic becomes, under the title of each pane. */
  hint: string;
  /** Says what a picked tag becomes. Defaults to hint. */
  tagHint?: string;
  /** Names what one picked value of a topic adds, such as "field". */
  noun: string;
  /** Names what one picked tag adds. Defaults to noun. */
  tagNoun?: string;
  /** Each pick is a whole topic, and a topic that the task holds takes none. */
  whole?: boolean;
  /** A pick of a topic that the task does not read takes its index too. */
  indexed?: boolean;
  /**
   * Each entry from a topic sends one message, which a field under the topics picks. An
   * add that holds a topic before the pick shows the error at that field.
   */
  formatRequired?: boolean;
  /** @returns what the picks add to a task that holds the entries. */
  create: (picks: Picks, entries: HeldEntry[]) => E[];
  /** @returns what a topic added by name adds. Absent, the dialog lists no topics. */
  createTopic?: (topic: string, picks: Picks) => E;
  /** @returns what a tag added by name adds. Absent, the dialog lists no tags. */
  createTag?: () => E;
}

export interface Params {
  device: Device.Device;
  /** What the task holds now. */
  entries: HeldEntry[];
}

export type DiscoverProps<E> = Modals.ContentProps<Params, E[]> & {
  kind: Kind<E>;
};

export const Discover = <E,>({ device, entries, close, kind }: DiscoverProps<E>) => {
  const { createTopic, createTag, noun, tagNoun = noun, hint, tagHint = hint } = kind;
  const [mode, setMode] = useState<Mode>(createTopic == null ? "sparkplug" : "topics");
  const [topicPicks, setTopicPicks] = useState<TopicPicks>(() => new Map());
  const [tagPicks, setTagPicks] = useState<TagPicks>(() => new Map());
  const [format, setFormat] = useState<Format | null>(null);
  const [refused, setRefused] = useState(false);
  const picks: Picks = { topics: topicPicks, tags: tagPicks, format };
  const formatError = refused && format == null ? FORMAT_REQUIRED : undefined;

  const add = (extra: () => E[], extraTopics = 0) => {
    const topicCount = topicPicks.size + extraTopics;
    if (kind.formatRequired === true && format == null && topicCount > 0) {
      setMode("topics");
      setRefused(true);
      return;
    }
    close([...kind.create(picks, entries), ...extra()]);
  };

  const entryCount =
    kind.whole === true
      ? topicPicks.size
      : math.sum([...topicPicks.values()].map(countOf));
  const parts: string[] = [];
  const topics = pluralize("topic", topicPicks.size, true);
  if (kind.whole === true && entryCount > 0) parts.push(topics);
  else if (entryCount > 0)
    parts.push(`${pluralize("value", entryCount, true)} from ${topics}`);
  if (tagPicks.size > 0) parts.push(pluralize("Sparkplug B tag", tagPicks.size, true));
  const summary = parts.length === 0 ? null : `${parts.join(" and ")} selected`;

  return (
    <ICD.Picker.Root value={mode} className={CSS.B("mqtt-discover")}>
      <Modal.Header icon={<Icon.Logo.MQTT />}>{[kind.title, device.name]}</Modal.Header>
      <ICD.Picker.Bar>
        {createTopic != null && createTag != null && (
          <Select.Buttons<Mode> value={mode} onChange={setMode}>
            <Select.Item<Mode> itemKey="topics">JSON topics</Select.Item>
            <Select.Item<Mode> itemKey="sparkplug">Sparkplug B</Select.Item>
          </Select.Buttons>
        )}
      </ICD.Picker.Bar>
      {createTopic != null && (
        <ICD.Picker.Frame itemKey="topics" hint={hint}>
          <TopicPicker
            device={device}
            entries={entries}
            whole={kind.whole}
            indexed={kind.indexed}
            value={topicPicks}
            onChange={setTopicPicks}
            onAddByName={(topic) => add(() => [createTopic(topic, picks)], 1)}
          >
            {kind.formatRequired === true && (
              <Flex.Box y empty className={CSS.BE("mqtt-discover", "format")}>
                <ICD.MessageFormatField
                  icdKey={format?.icd}
                  message={format?.message.key}
                  onChange={(icdKey, message) => setFormat({ icd: icdKey, message })}
                  helpText={formatError}
                  status={formatError == null ? undefined : "error"}
                  padHelpText
                />
              </Flex.Box>
            )}
          </TopicPicker>
        </ICD.Picker.Frame>
      )}
      {createTag != null && (
        <ICD.Picker.Frame itemKey="sparkplug" hint={tagHint}>
          <TagPicker
            device={device}
            entries={entries}
            value={tagPicks}
            onChange={setTagPicks}
            onAddByName={() => add(() => [createTag()])}
          />
        </ICD.Picker.Frame>
      )}
      <ICD.Picker.Footer
        counts={[
          [noun, entryCount],
          [tagNoun, tagPicks.size],
        ]}
        idle={mode === "sparkplug" ? tagNoun : noun}
        onAdd={() => add(() => [])}
        onCancel={() => close()}
      >
        {summary != null && (
          <Text.Text level="small" color={10}>
            {summary}
          </Text.Text>
        )}
      </ICD.Picker.Footer>
    </ICD.Picker.Root>
  );
};
