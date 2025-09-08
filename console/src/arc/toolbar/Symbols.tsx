// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/toolbar/Symbols.css";

import {
  Arc,
  Component,
  CSS as PCSS,
  Divider,
  Flex,
  Haul,
  List,
  Select,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { addElement } from "@/arc/slice";
import { CSS } from "@/css";

export interface SymbolsProps {
  group: string;
  layoutKey: string;
}

export const Group = ({ group, layoutKey }: SymbolsProps): ReactElement => {
  const dispatch = useDispatch();
  const theme = Theming.use();

  const groupRegistry = useMemo(() => Arc.REGISTRY[group], [group]);
  const symbols = useMemo(() => Object.values(groupRegistry.symbols), [groupRegistry]);

  const handleAddElement = useCallback(
    (variant: string) => {
      const spec = groupRegistry.symbols[variant];
      const initialProps = spec.defaultProps(theme);
      dispatch(
        addElement({
          key: layoutKey,
          elKey: id.create(),
          node: {
            zIndex: spec.zIndex,
          },
          props: {
            key: variant,
            ...initialProps,
          },
        }),
      );
    },
    [dispatch, layoutKey, theme],
  );

  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Diagram-Elements",
    key: "symbols",
  });

  const handleDragStart = useCallback(
    (key: string) => {
      startDrag([{ type: "arc-element", key }]);
    },
    [startDrag],
  );

  return (
    <Flex.Box
      x
      className={CSS(
        CSS.B("arc-symbols"),
        PCSS.BE("symbol", "container"),
        PCSS.M("editable"),
      )}
      wrap
      empty
      grow
    >
      {symbols.map((p) => (
        <SymbolsButton
          key={p.key}
          symbolSpec={p}
          onClick={() => handleAddElement(p.key)}
          theme={theme}
          startDrag={handleDragStart}
          onDragEnd={onDragEnd}
        />
      ))}
    </Flex.Box>
  );
};

interface SymbolsButtonProps extends PropsWithChildren, Flex.BoxProps {
  symbolSpec: Arc.Spec<any>;
  theme: Theming.Theme;
  startDrag: (key: string) => void;
}

const SymbolsButton = ({
  children,
  symbolSpec: { name, key, Preview, defaultProps },
  theme,
  startDrag,
  onDragEnd,
  ...rest
}: SymbolsButtonProps): ReactElement => {
  const defaultProps_ = useMemo(() => defaultProps(theme), [defaultProps, theme]);

  return (
    <Flex.Box
      className={CSS(CSS.BE("arc-symbols", "button"))}
      justify="between"
      align="center"
      gap="tiny"
      draggable
      {...rest}
      onDragStart={() => startDrag(key)}
      onDragEnd={onDragEnd}
    >
      <Text.Text level="small">{name}</Text.Text>
      <Flex.Box className="preview-wrapper" align="center" justify="center">
        <Preview {...defaultProps_} scale={0.75} />
      </Flex.Box>
    </Flex.Box>
  );
};

const GROUP_LIST_DATA = Object.keys(Arc.REGISTRY);

const groupListItem = Component.renderProp((props: List.ItemProps<string>) => {
  const group = useMemo(() => Arc.REGISTRY[props.itemKey], [props.itemKey]);
  const selectProps = Select.useItemState(props.itemKey);
  return (
    <List.Item
      {...props}
      {...selectProps}
      level="p"
      textColor={10}
      style={{ minHeight: "4.5rem", padding: "0 2rem" }}
    >
      {group.icon}
      {group.name}
    </List.Item>
  );
});

export const Symbols = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const [selectedGroup, setSelectedGroup] = useState<string>("basic");
  return (
    <Flex.Box x empty full>
      <Select.Frame<string, Arc.Group>
        data={GROUP_LIST_DATA}
        value={selectedGroup}
        onChange={setSelectedGroup}
      >
        <List.Items<string, Arc.Group>>{groupListItem}</List.Items>
      </Select.Frame>
      <Divider.Divider y />
      <Group group={selectedGroup} layoutKey={layoutKey} />
    </Flex.Box>
  );
};
