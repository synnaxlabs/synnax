// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/slate/toolbar/Symbols.css";

import {
  Align,
  CSS as PCSS,
  Divider,
  Haul,
  List,
  Slate,
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

import { CSS } from "@/css";
import { addElement } from "@/slate/slice";

export interface SymbolsProps {
  group: string;
  layoutKey: string;
}

export const Group = ({ group, layoutKey }: SymbolsProps): ReactElement => {
  const dispatch = useDispatch();
  const theme = Theming.use();

  const groupRegistry = useMemo(() => Slate.REGISTRY[group], [group]);
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
      startDrag([{ type: "slate-element", key }]);
    },
    [startDrag],
  );

  return (
    <List.List data={symbols}>
      <List.Core<string, Slate.Spec<any>>
        x
        className={CSS(
          CSS.B("slate-symbols"),
          PCSS.BE("symbol", "container"),
          PCSS.M("editable"),
        )}
        wrap
        empty
      >
        {(p) => (
          <SymbolsButton
            key={p.key}
            symbolSpec={p.entry}
            onClick={() => handleAddElement(p.entry.key)}
            theme={theme}
            startDrag={handleDragStart}
            onDragEnd={onDragEnd}
          />
        )}
      </List.Core>
    </List.List>
  );
};

interface SymbolsButtonProps extends PropsWithChildren, Align.SpaceProps {
  symbolSpec: Slate.Spec<any>;
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
    <Align.Space
      className={CSS(CSS.BE("slate-symbols", "button"))}
      justify="spaceBetween"
      align="center"
      size="tiny"
      draggable
      {...rest}
      onDragStart={() => startDrag(key)}
      onDragEnd={onDragEnd}
    >
      <Text.Text level="small">{name}</Text.Text>
      <Align.Space className="preview-wrapper" align="center" justify="center">
        <Preview {...defaultProps_} scale={0.75} />
      </Align.Space>
    </Align.Space>
  );
};

const GROUP_LIST_DATA = Object.values(Slate.REGISTRY);

export const Symbols = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const [selectedGroup, setSelectedGroup] = useState<string>("basic");

  return (
    <Align.Space x empty grow style={{ height: "100%" }}>
      <Align.Space x empty style={{ width: 150 }}>
        <List.List<string, Slate.Group> data={GROUP_LIST_DATA}>
          <List.Selector<string, Slate.Group>
            allowMultiple={false}
            value={selectedGroup}
            onChange={setSelectedGroup}
          >
            <List.Core<string, Slate.Group> style={{ width: "100%" }}>
              {({ key, ...rest }) => (
                <List.ItemFrame key={key} {...rest} style={{ padding: "4px 2rem" }}>
                  <Text.WithIcon level="p" startIcon={rest.entry.icon} size="medium">
                    {rest.entry.name}
                  </Text.WithIcon>
                </List.ItemFrame>
              )}
            </List.Core>
          </List.Selector>
        </List.List>
      </Align.Space>
      <Divider.Divider y />
      <Group group={selectedGroup} layoutKey={layoutKey} />
    </Align.Space>
  );
};
