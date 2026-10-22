// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/device/Browser.css";

import { DisconnectedError, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import {
  type BrowseResult,
  browseTopics,
  type KeyedTopic,
} from "@/feature/mqtt/device/browse";
import { SparkplugBrowser } from "@/feature/mqtt/device/SparkplugBrowser";
import { type Device } from "@/feature/mqtt/device/types";
import { type BrowsedTopic } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";

export const HAUL_TYPE = "mqtt";

export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, BrowsedTopic>;

export const createHaulItem = ({
  topic,
  payload,
  retained,
}: BrowsedTopic): HaulItem => ({
  type: HAUL_TYPE,
  key: topic,
  data: { topic, payload, retained },
});

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter(isHaulItem);

export const canDropHaulItem = Haul.canDropOfType<HaulItem>(HAUL_TYPE);

export interface BrowserProps {
  device: Device;
}

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const topic = List.useItem<string, KeyedTopic>(itemKey);
  const { getState } = Select.useContext<string>();
  const { getItem } = List.useUtilContext<string, KeyedTopic>();
  const { startDrag } = Haul.useDrag({ type: HAUL_TYPE, key: itemKey });
  const handleDragStart = useCallback(() => {
    if (topic == null) return;
    const selected = array.toArray(getState().value);
    if (getItem != null && selected.includes(itemKey))
      startDrag(getItem(selected).map(createHaulItem));
    else startDrag([createHaulItem(topic)]);
  }, [startDrag, topic, getState, getItem, itemKey]);
  if (topic == null) return null;
  return (
    <Select.Item {...props} draggable onDragStart={handleDragStart} y gap="tiny">
      <Flex.Box x justify="between" align="center">
        <Text.Text level="small" weight={500} color={10} overflow="ellipsis">
          {topic.topic}
        </Text.Text>
        {topic.retained && <Tag.Tag size="small">Retained</Tag.Tag>}
      </Flex.Box>
      <Text.Text level="small" color={9} overflow="ellipsis">
        {topic.payload}
      </Text.Text>
    </Select.Item>
  );
});

const DEFAULT_FILTER = "#";

const FILTER_INPUT_PROPS = { placeholder: DEFAULT_FILTER } as const;

const EMPTY_CONTENT = (
  <Text.Text center status="disabled" className={CSS.BE("mqtt-browser", "empty")}>
    No topics. Browse the broker to list the topics that publish now.
  </Text.Text>
);

const TopicBrowser = ({ device }: BrowserProps) => {
  const [filter, setFilter] = useState(DEFAULT_FILTER);
  const [result, setResult] = useState<BrowseResult>({ topics: [], truncated: false });
  const [selected, setSelected] = useState<string[]>([]);
  const [stat, setStat] = useState<status.Status | null>(null);
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const browse = useCallback(() => {
    setStat(status.create({ variant: "loading", message: "Browsing MQTT topics" }));
    handleError(async () => {
      try {
        if (client == null) throw new DisconnectedError();
        setResult(await browseTopics(client, device, filter));
      } catch (e) {
        setStat(status.fromException(e, "Failed to browse MQTT topics"));
        return;
      }
      setSelected([]);
      setStat(status.create({ variant: "success", message: "Browsed MQTT topics" }));
    }, "Failed to browse MQTT topics");
  }, [client, device, filter, handleError]);
  const listProps = List.useStaticData<string, KeyedTopic>({ data: result.topics });
  const loading = stat?.variant === "loading";
  let content: ReactElement;
  if (stat?.variant === "error") content = <Status.Summary center status={stat} />;
  else if (loading)
    content = (
      <Flex.Box center>
        <Icon.Loading className={CSS.BE("mqtt-browser", "loading-icon")} color={9} />
      </Flex.Box>
    );
  else
    content = (
      <Select.Frame<string, KeyedTopic>
        multiple
        data={listProps.data}
        getItem={listProps.getItem}
        value={selected}
        onChange={setSelected}
        replaceOnSingle
      >
        <List.Scroll full="y">
          <List.Items<string, KeyedTopic> emptyContent={EMPTY_CONTENT}>
            {listItem}
          </List.Items>
        </List.Scroll>
      </Select.Frame>
    );
  return (
    <>
      <Flex.Box x className={CSS.BE("mqtt-browser", "controls")}>
        <Input.Text value={filter} onChange={setFilter} {...FILTER_INPUT_PROPS} />
        <Button.Button onClick={browse} disabled={loading} variant="filled">
          Browse
        </Button.Button>
      </Flex.Box>
      {result.truncated && !loading && (
        <Text.Text
          level="small"
          status="warning"
          className={CSS.BE("mqtt-browser", "note")}
        >
          The broker delivered more topics than the list holds. Use a narrower filter.
        </Text.Text>
      )}
      {content}
    </>
  );
};

type Mode = "topics" | "sparkplug";

export const Browser = ({ device }: BrowserProps) => {
  const [mode, setMode] = useState<Mode>("topics");
  return (
    <Flex.Box empty className={CSS.B("mqtt-browser")}>
      <Header.Header>
        <Header.Title weight={500} color={10}>
          Browser
        </Header.Title>
      </Header.Header>
      <Select.Buttons<Mode>
        value={mode}
        onChange={setMode}
        className={CSS.BE("mqtt-browser", "mode")}
      >
        <Select.Item<Mode> itemKey="topics" grow justify="center">
          Topics
        </Select.Item>
        <Select.Item<Mode> itemKey="sparkplug" grow justify="center">
          Sparkplug B
        </Select.Item>
      </Select.Buttons>
      {mode === "topics" ? (
        <TopicBrowser device={device} />
      ) : (
        <SparkplugBrowser device={device} />
      )}
    </Flex.Box>
  );
};
