// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/toolbar/Symbols.css";

import { Align, Haul, Input, Schematic, Text, Theming } from "@synnaxlabs/pluto";
import { List } from "@synnaxlabs/pluto/list";
import { nanoid } from "nanoid/non-secure";
import {
  type ComponentPropsWithoutRef,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
} from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { addElement } from "@/schematic/slice";

const LIST_DATA = Object.values(Schematic.SYMBOLS);

export const Symbols = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const dispatch = useDispatch();
  const theme = Theming.use();

  const handleAddElement = useCallback(
    (variant: Schematic.Variant) => {
      const spec = Schematic.SYMBOLS[variant];
      const initialProps = spec.defaultProps(theme);
      dispatch(
        addElement({
          key: layoutKey,
          elKey: nanoid(),
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

  return (
    <List.List data={LIST_DATA}>
      <Align.Space style={{ padding: "1rem", borderBottom: "var(--pluto-border)" }}>
        <List.Filter>
          {(p) => <Input.Text {...p} placeholder="Type to search..." size="small" />}
        </List.Filter>
      </Align.Space>
      <List.Core<string, Schematic.Spec<any>>
        direction="x"
        className={CSS.B("schematic-symbols")}
        wrap
      >
        {(p) => (
          <SymbolsButton
            key={p.key}
            el={p.entry}
            onClick={() => handleAddElement(p.entry.key)}
            theme={theme}
          />
        )}
      </List.Core>
    </List.List>
  );
};

interface SymbolsButtonProps
  extends PropsWithChildren,
    ComponentPropsWithoutRef<"button"> {
  el: Schematic.Spec<any>;
  theme: Theming.Theme;
}

const SymbolsButton = ({
  children,
  el: { name, key, Preview, defaultProps },
  theme,
  ...props
}: SymbolsButtonProps): ReactElement => {
  const { startDrag, ...dragProps } = Haul.useDrag({
    type: "Diagram-Elements",
    key: name,
  });

  const handleDragStart = useCallback(() => {
    startDrag([{ type: "schematic-element", key }]);
  }, [key]);

  return (
    // @ts-expect-error - generic elements
    <Align.Space
      el="button"
      className={CSS.BE("schematic-symbols", "button")}
      justify="spaceBetween"
      align="center"
      draggable
      {...props}
      {...dragProps}
      onDragStart={handleDragStart}
    >
      <Text.Text level="small">{name}</Text.Text>
      <Align.Space className="preview-wrapper" align="center" justify="center">
        <Preview {...defaultProps(theme)} scale={0.8} />
      </Align.Space>
    </Align.Space>
  );
};
