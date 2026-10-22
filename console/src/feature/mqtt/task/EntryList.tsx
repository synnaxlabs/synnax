// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useCallback } from "react";

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
  /** Takes the entries built from a drop; a topic or tag the list holds is skipped. */
  onAdd: (items: E[]) => void;
}

/** Makes an element a target for topics and tags dragged from the Browser. */
export const useEntryDrop = <E extends Item>({
  path,
  create,
  createSparkplug,
  onAdd,
}: UseEntryDropProps<E>): Partial<Haul.UseDropReturn> => {
  const { get } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const handleDrop = useCallback(
    ({ items: dropped }: Haul.OnDropProps): Haul.Item[] => {
      const present = new Set(
        get<E[]>(path).value.map((item) =>
          item.type === "plain" ? item.topic : tagIdentity(item),
        ),
      );
      const topics = filterHaulItems(dropped);
      const tags = filterSparkplugHaulItems(dropped);
      const added = [
        ...topics
          .filter(({ data }) => !present.has(data.topic))
          .map(({ data }) => create(data)),
        ...tags
          .filter(({ data }) => !present.has(tagIdentity(data)))
          .map(({ data }) => createSparkplug(data)),
      ];
      if (added.length > 0) onAdd(added);
      return [...topics, ...tags];
    },
    [get, path, create, createSparkplug, onAdd],
  );
  const dropProps = Haul.useDrop({ type: HAUL_TYPE, canDrop, onDrop: handleDrop });
  // The browser hides in preview, but the browser of a second tab can still source
  // drags, so the drop target goes inert too.
  return isPreview ? {} : dropProps;
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
