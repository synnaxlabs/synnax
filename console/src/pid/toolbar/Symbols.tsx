// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ComponentPropsWithoutRef,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
} from "react";

import { Align, Text, PID, Theming, Haul, Input } from "@synnaxlabs/pluto";
import { List } from "@synnaxlabs/pluto/list";
import { nanoid } from "nanoid/non-secure";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { addElement } from "@/pid/slice";

import "@/pid/toolbar/Symbols.css";

const LIST_DATA = Object.values(PID.SYMBOLS);

export const Symbols = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const dispatch = useDispatch();
  const theme = Theming.use();

  const handleAddElement = useCallback(
    (variant: PID.Variant) => {
      const spec = PID.SYMBOLS[variant];
      const initialProps = spec.defaultProps(theme);
      dispatch(
        addElement({
          layoutKey,
          key: nanoid(),
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
      <List.Core<string, PID.Spec<any>>
        direction="x"
        className={CSS.B("pid-symbols")}
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
  el: PID.Spec<any>;
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
    startDrag([{ type: "pid-element", key }]);
  }, [key]);

  return (
    // @ts-expect-error - generic elements
    <Align.Space
      el="button"
      className={CSS.BE("pid-symbols", "button")}
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
