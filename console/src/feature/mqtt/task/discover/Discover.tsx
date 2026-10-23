// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/discover/Discover.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { useState } from "react";

import { type Device } from "@/feature/mqtt/device/types";
import { plural } from "@/feature/mqtt/task/discover/Picker";
import { TagPicker, type TagPicks } from "@/feature/mqtt/task/discover/Tags";
import { TopicPicker, type TopicPicks } from "@/feature/mqtt/task/discover/Topics";
import { createEntry, createSparkplugEntry } from "@/feature/mqtt/task/entries";
import { type PlainReadEntry, type ReadEntry } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

type Mode = "topics" | "sparkplug";

interface Params {
  device: Device;
  /** The entries the task reads now. */
  entries: ReadEntry[];
}

const Discover = ({
  device,
  entries,
  close,
}: Modals.ContentProps<Params, ReadEntry[]>) => {
  const [mode, setMode] = useState<Mode>("topics");
  // The Sparkplug B tab listens only once the user opens it.
  const [sparkplugOpened, setSparkplugOpened] = useState(false);
  const [topicPicks, setTopicPicks] = useState<TopicPicks>(() => new Map());
  const [tagPicks, setTagPicks] = useState<TagPicks>(() => new Map());

  const handleMode = (next: Mode) => {
    setMode(next);
    if (next === "sparkplug") setSparkplugOpened(true);
  };

  const picked = (): ReadEntry[] => [
    ...[...topicPicks].map(([topic, { leaves, pointers }]): ReadEntry => {
      const fields = leaves
        .filter((l) => pointers.has(l.pointer))
        .map((l) => ({ ...l, disabled: false }));
      const existing = entries.find(
        (e): e is PlainReadEntry => e.type === "plain" && e.topic === topic,
      );
      return existing == null
        ? { ...createEntry(), topic, fields }
        : { ...existing, fields: [...existing.fields, ...fields] };
    }),
    ...[...tagPicks.values()].map((t) => createSparkplugEntry(t)),
  ];

  const values = [...topicPicks.values()].reduce((n, p) => n + p.pointers.size, 0);
  const channels = values + tagPicks.size;
  const parts: string[] = [];
  if (values > 0)
    parts.push(`${plural(values, "value")} from ${plural(topicPicks.size, "topic")}`);
  if (tagPicks.size > 0) parts.push(plural(tagPicks.size, "Sparkplug B tag"));
  const summary =
    parts.length === 0
      ? "Select the values to record. Each one becomes a channel."
      : `${parts.join(" and ")} selected`;

  const tabs = (
    <Select.Buttons<Mode> value={mode} onChange={handleMode}>
      <Select.Item<Mode> itemKey="topics">JSON topics</Select.Item>
      <Select.Item<Mode> itemKey="sparkplug">Sparkplug B</Select.Item>
    </Select.Buttons>
  );

  return (
    <Modal.Frame className={CSS.B("mqtt-discover")}>
      <Modal.Header icon={<Icon.Logo.MQTT />}>{["Add data", device.name]}</Modal.Header>
      <TopicPicker
        device={device}
        entries={entries}
        hidden={mode !== "topics"}
        tabs={tabs}
        picks={topicPicks}
        onPicksChange={setTopicPicks}
        onAddByName={(topic) => close([...picked(), { ...createEntry(), topic }])}
      />
      {sparkplugOpened && (
        <TagPicker
          device={device}
          entries={entries}
          hidden={mode !== "sparkplug"}
          tabs={tabs}
          picks={tagPicks}
          onPicksChange={setTagPicks}
          onAddByName={() => close([...picked(), createSparkplugEntry()])}
        />
      )}
      <Modal.Footer>
        <Nav.Bar.Start>
          <Text.Text level="small" color={channels > 0 ? 10 : 9}>
            {summary}
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End x align="center" gap="small">
          <Button.Button variant="outlined" onClick={() => close()}>
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            disabled={channels === 0}
            onClick={() => close(picked())}
          >
            {channels === 0 ? "Add channels" : `Add ${plural(channels, "channel")}`}
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a dialog that lists what the broker publishes and lets the user pick values. It
 * resolves with the entries picked, or null when the user closes it. A picked topic the
 * task reads already comes back as its entry, under the same key, with the new fields
 * added.
 */
export const useDiscoverModal = Modals.createPrompt<ReadEntry[], Params>(Discover);
