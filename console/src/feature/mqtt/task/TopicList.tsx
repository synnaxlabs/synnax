// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  type Component,
  Flex,
  Form as PForm,
  Haul,
  Header,
  Icon,
  List,
  Menu,
  Select,
} from "@synnaxlabs/pluto";
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
import { ContextMenu } from "@/feature/mqtt/task/ContextMenu";
import { type BrowsedTopic, type SparkplugTagID } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Task } from "@/platform/task";

type Item =
  | { key: string; type: "plain"; topic: string }
  | ({ key: string; type: "sparkplug" } & SparkplugTagID);

const tagIdentity = ({ group, edgeNode, device, tag }: SparkplugTagID): string =>
  JSON.stringify([group, edgeNode, device, tag]);

const canDrop: Haul.CanDrop = (state) =>
  canDropHaulItem(state) || canDropSparkplugHaulItem(state);

export interface TopicListProps<E extends Item> {
  path: string;
  title: string;
  noun: string;
  selected: string[];
  onSelect: (keys: string[]) => void;
  create: (topic?: BrowsedTopic) => E;
  createSparkplug: (tag?: SparkplugHaulTag) => E;
  duplicate: (item: E) => E;
  onRename?: (key: string) => void;
  children: Component.RenderProp<List.ItemProps<string>>;
}

export const TopicList = <E extends Item>({
  path,
  title,
  noun,
  selected,
  onSelect,
  create,
  createSparkplug,
  duplicate,
  onRename,
  children,
}: TopicListProps<E>) => {
  const { data, push, remove } = PForm.useFieldList<string, E>(path);
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();

  const add = useCallback(
    (item: E) => {
      push(item);
      onSelect([item.key]);
    },
    [push, onSelect],
  );
  const handleAdd = useCallback(() => add(create()), [add, create]);
  const handleAddSparkplug = useCallback(
    () => add(createSparkplug()),
    [add, createSparkplug],
  );

  const handleRemove = useCallback(
    (keys: string[]) => {
      remove(keys);
      onSelect([]);
    },
    [remove, onSelect],
  );

  const handleDuplicate = useCallback(
    (keys: string[]) => {
      const duplicated = ctx
        .get<E[]>(path)
        .value.filter(({ key }) => keys.includes(key))
        .map(duplicate);
      push(duplicated);
      if (duplicated.length > 0) onSelect([duplicated[0].key]);
    },
    [ctx, path, duplicate, push, onSelect],
  );

  const handleDrop = useCallback(
    ({ items: dropped }: Haul.OnDropProps): Haul.Item[] => {
      const present = new Set(
        ctx
          .get<E[]>(path)
          .value.map((item) =>
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
      push(added);
      if (added.length > 0) onSelect([added[0].key]);
      return [...topics, ...tags];
    },
    [ctx, path, create, createSparkplug, push, onSelect],
  );

  const dropProps = Haul.useDrop({ type: HAUL_TYPE, canDrop, onDrop: handleDrop });
  // The browser hides in preview, but the browser of a second tab can still source
  // drags, so the drop target goes inert too.
  const haulProps = isPreview ? {} : dropProps;

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    (p: Menu.ContextMenuMenuProps) => (
      <ContextMenu
        keys={p.keys}
        disablePath={path}
        onRemove={handleRemove}
        onDuplicate={handleDuplicate}
        onRename={onRename}
      />
    ),
    [path, handleRemove, handleDuplicate, onRename],
  );

  return (
    <Flex.Box className={CSS.B("topic-list")} y empty>
      <Header.Header>
        <Header.Title weight={500} color={10}>
          {title}
        </Header.Title>
        {!isPreview && (
          <Header.Actions>
            <Button.Button
              onClick={handleAdd}
              variant="filled"
              tooltip={`Add ${noun}`}
              size="small"
            >
              <Icon.Add />
            </Button.Button>
            <Button.Button
              onClick={handleAddSparkplug}
              variant="filled"
              tooltip="Add Sparkplug B tag"
              size="small"
            >
              <Icon.Variable />
            </Button.Button>
          </Header.Actions>
        )}
      </Header.Header>
      <Menu.ContextMenu {...menuProps} {...haulProps} menu={menuRenderProp}>
        <Select.Frame<string, E>
          multiple
          data={data}
          value={selected}
          onChange={onSelect}
          replaceOnSingle
          allowNone={false}
          autoSelectOnNone
        >
          <List.Items<string, E>
            full="y"
            className={menuProps.className}
            onContextMenu={menuProps.open}
            emptyContent={
              <Empty.Action
                message={`No ${title.toLowerCase()}`}
                action={isPreview ? undefined : `Add ${noun}`}
                onClick={handleAdd}
              />
            }
            {...haulProps}
          >
            {children}
          </List.Items>
        </Select.Frame>
      </Menu.ContextMenu>
    </Flex.Box>
  );
};
