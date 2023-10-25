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

import { Align, Text, PIDElement, Theming, Haul, Input, List } from "@synnaxlabs/pluto";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { addElement } from "@/pid/slice";

import "@/pid/toolbar/Elements.css";

const LIST_DATA = Object.values(PIDElement.REGISTRY).map((el) => ({
  key: el.type,
  ...el,
}));

export const Elements = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const dispatch = useDispatch();
  const theme = Theming.use();

  const handleAddElement = useCallback(
    (type: string) => {
      const spec = PIDElement.REGISTRY[type];
      const initialProps = spec.initialProps(theme);
      dispatch(
        addElement({
          layoutKey,
          key: nanoid(),
          props: {
            type,
            ...initialProps,
          },
          node: { zIndex: spec.zIndex },
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
      <List.Core direction="x" className={CSS.B("pid-elements")}>
        {(p) => (
          <ElementsButton
            key={p.entry.type}
            el={p.entry}
            onClick={() => handleAddElement(p.entry.type)}
            theme={theme}
          />
        )}
      </List.Core>
    </List.List>
  );
};

interface ElementsButtonProps
  extends PropsWithChildren,
    ComponentPropsWithoutRef<"button"> {
  el: PIDElement.Spec;
  theme: Theming.Theme;
}

const ElementsButton = ({
  children,
  el: { title, type, Preview, initialProps },
  theme,
  ...props
}: ElementsButtonProps): ReactElement => {
  const { startDrag, ...dragProps } = Haul.useDrag({
    type: "PID-Elements",
    key: title,
  });

  const handleDragStart = useCallback(() => {
    startDrag([{ type: "pid-element", key: type }]);
  }, [type]);

  return (
    <>
      {/* @ts-expect-error */}
      <Align.Space
        el="button"
        className={CSS.BE("pid-elements", "button")}
        justify="spaceBetween"
        align="center"
        draggable
        {...props}
        {...dragProps}
        onDragStart={handleDragStart}
      >
        <Text.Text level="p" color="var(--pluto-gray-p0)">
          {title}
        </Text.Text>
        <Preview {...initialProps(theme)} />
      </Align.Space>
    </>
  );
};
