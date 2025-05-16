// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/stage/toolbar/Symbols.css";

import {
  Align,
  CSS as PCSS,
  Haul,
  Input,
  List,
  Stage,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { addElement } from "@/stage/slice";

const LIST_DATA = Object.values(Stage.SYMBOLS);

export interface SymbolsProps {
  layoutKey: string;
}

export const Symbols = ({ layoutKey }: SymbolsProps): ReactElement => {
  const dispatch = useDispatch();
  const theme = Theming.use();

  const handleAddElement = useCallback(
    (variant: Stage.Variant) => {
      const spec = Stage.SYMBOLS[variant];
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
      startDrag([{ type: "stage-element", key }]);
    },
    [startDrag],
  );

  return (
    <List.List data={LIST_DATA}>
      <Align.Space style={{ padding: "1rem", borderBottom: "var(--pluto-border)" }}>
        <List.Filter>
          {(p) => <Input.Text {...p} placeholder="Type to search..." size="small" />}
        </List.Filter>
      </Align.Space>
      <List.Core<string, Stage.Spec<any>>
        x
        className={CSS(
          CSS.B("stage-symbols"),
          PCSS.BE("symbol", "container"),
          PCSS.M("editable"),
        )}
        wrap
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
  symbolSpec: Stage.Spec<any>;
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
      className={CSS(CSS.BE("stage-symbols", "button"))}
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
