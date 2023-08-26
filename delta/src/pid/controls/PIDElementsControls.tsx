// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ComponentPropsWithoutRef,
  PropsWithChildren,
  ReactElement,
  useCallback,
} from "react";

import { Align, Text, PIDElement } from "@synnaxlabs/pluto";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";

import { addPIDelement } from "../store/slice";

import "@/pid/controls/PIDElementsControls.css";

export const PIDElements = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const dispatch = useDispatch();

  const handleAddElement = useCallback(
    (type: string) =>
      dispatch(
        addPIDelement({
          layoutKey,
          key: nanoid(),
          props: {
            type,
            ...PIDElement.REGISTRY[type].initialProps,
          },
        })
      ),
    [dispatch, layoutKey]
  );

  return (
    <Align.Space className={CSS.B("pid-elements")} direction="x" wrap>
      {Object.entries(PIDElement.REGISTRY).map(([type, el]) => (
        <PIDElementsButton key={type} el={el} onClick={() => handleAddElement(type)} />
      ))}
    </Align.Space>
  );
};

interface PIDElementsButtonProps
  extends PropsWithChildren,
    ComponentPropsWithoutRef<"button"> {
  el: PIDElement.Spec;
}

const PIDElementsButton = ({
  children,
  el: { title, Preview },
  ...props
}: PIDElementsButtonProps): ReactElement => {
  return (
    <>
      {/* @ts-expect-error */}
      <Align.Space
        el="button"
        className={CSS.BE("pid-elements", "button")}
        justify="center"
        align="center"
        {...props}
      >
        <Text.Text level="p" color="var(--pluto-gray-p0)">
          {title}
        </Text.Text>
        <Preview />
      </Align.Space>
    </>
  );
};
