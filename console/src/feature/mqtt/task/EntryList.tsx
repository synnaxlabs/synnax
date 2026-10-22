// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { useCallback, useMemo } from "react";

import {
  canDropHaulItem,
  filterHaulItems,
  HAUL_TYPE,
} from "@/feature/mqtt/device/Browser";
import {
  canDropSparkplugHaulItem,
  filterSparkplugHaulItems,
  type SparkplugHaulTag,
} from "@/feature/mqtt/device/SparkplugBrowser";
import { type BrowsedTopic, type SparkplugTagID } from "@/feature/mqtt/task/types";
import { Button as PlatformButton } from "@/platform/button";
import { Task } from "@/platform/task";

type Item =
  | { key: string; type: "plain"; topic: string }
  | ({ key: string; type: "sparkplug" } & SparkplugTagID);

const tagIdentity = ({ group, edgeNode, device, tag }: SparkplugTagID): string =>
  JSON.stringify([group, edgeNode, device, tag]);

const canDrop: Haul.CanDrop = (state) =>
  canDropHaulItem(state) || canDropSparkplugHaulItem(state);

export interface UseEntryDropProps<E extends Item> {
  path: string;
  create: (topic: BrowsedTopic) => E;
  createSparkplug: (tag: SparkplugHaulTag) => E;
}

/**
 * Builds entries from the topics and tags dragged from the Browser. A topic or tag the
 * list holds builds nothing.
 */
export const useEntryDrop = <E extends Item>({
  path,
  create,
  createSparkplug,
}: UseEntryDropProps<E>): Task.Views.Drop<E> => {
  const { get } = PForm.useContext();
  const onDrop = useCallback(
    ({ items }: Haul.OnDropProps) => {
      const present = new Set(
        get<E[]>(path).value.map((item) =>
          item.type === "plain" ? item.topic : tagIdentity(item),
        ),
      );
      const topics = filterHaulItems(items);
      const tags = filterSparkplugHaulItems(items);
      return {
        dropped: [...topics, ...tags],
        added: [
          ...topics
            .filter(({ data }) => !present.has(data.topic))
            .map(({ data }) => create(data)),
          ...tags
            .filter(({ data }) => !present.has(tagIdentity(data)))
            .map(({ data }) => createSparkplug(data)),
        ],
      };
    },
    [get, path, create, createSparkplug],
  );
  return useMemo(() => ({ type: HAUL_TYPE, canDrop, onDrop }), [onDrop]);
};

export interface AddEntryButtonsProps {
  onAdd: () => void;
  onAddSparkplug: () => void;
}

/** The create actions pinned under an entry list, one per kind of entry. */
export const AddEntryButtons = ({ onAdd, onAddSparkplug }: AddEntryButtonsProps) => {
  const isPreview = Task.useIsPreview();
  if (isPreview) return null;
  return (
    <>
      <PlatformButton.CreateListItem size="small" onClick={onAdd}>
        Add topic
      </PlatformButton.CreateListItem>
      <PlatformButton.CreateListItem size="small" onClick={onAddSparkplug}>
        Add Sparkplug B tag
      </PlatformButton.CreateListItem>
    </>
  );
};
