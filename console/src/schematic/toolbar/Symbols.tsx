// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/toolbar/Symbols.css";

import {
  Align,
  CSS as PCSS,
  Haul,
  Input,
  List,
  Schematic,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { addElement } from "@/schematic/slice";

const LIST_DATA = Object.values(Schematic.SYMBOLS);

export interface SymbolsProps {
  layoutKey: string;
}

export const Symbols = ({ layoutKey }: SymbolsProps): ReactElement => {
  const dispatch = useDispatch();
  const theme = Theming.use();

  const handleAddElement = useCallback(
    (variant: Schematic.Variant) => {
      const spec = Schematic.SYMBOLS[variant];
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
      startDrag([{ type: "schematic-element", key }]);
    },
    [startDrag],
  );

  const { data, useItem, retrieve } = List.useStaticData(LIST_DATA);
  const listProps = List.use({ data });
  const [search, setSearch] = useState("");
  return (
    <List.List data={data} useItem={useItem} {...listProps}>
      <Align.Space style={{ padding: "1rem", borderBottom: "var(--pluto-border)" }}>
        <Input.Text
          value={search}
          onChange={(v) => {
            setSearch(v);
            retrieve({ term: v });
          }}
          placeholder="Type to search..."
          size="small"
        />
      </Align.Space>
      <List.Items<Schematic.Variant, Schematic.Spec>
        className={CSS(
          CSS.B("schematic-symbols"),
          PCSS.BE("symbol", "container"),
          PCSS.M("editable"),
        )}
      >
        {(p) => (
          <ListItem
            {...p}
            key={p.key}
            onClick={() => handleAddElement(p.itemKey)}
            theme={theme}
            startDrag={handleDragStart}
            onDragEnd={onDragEnd}
          />
        )}
      </List.Items>
    </List.List>
  );
};

interface SymbolsButtonProps extends List.ItemProps<Schematic.Variant> {
  theme: Theming.Theme;
  startDrag: (key: string) => void;
}

const ListItem = ({
  onDragEnd,
  theme,
  translate: _,
  startDrag,
  itemKey,
}: SymbolsButtonProps): ReactElement | null => {
  const spec = List.useItem<Schematic.Variant, Schematic.Spec>(itemKey);
  const defaultProps_ = useMemo(() => spec?.defaultProps(theme), [spec, theme]);
  if (spec == null) return null;
  const { name, Preview } = spec;
  return (
    <Align.Space
      className={CSS(CSS.BE("schematic-symbols", "button"))}
      justify="spaceBetween"
      align="center"
      size="tiny"
      draggable
      onDragStart={() => startDrag(itemKey)}
      onDragEnd={onDragEnd}
    >
      <Text.Text level="small">{name}</Text.Text>
      <Align.Space className="preview-wrapper" align="center" justify="center">
        <Preview {...defaultProps_} scale={0.75} />
      </Align.Space>
    </Align.Space>
  );
};
