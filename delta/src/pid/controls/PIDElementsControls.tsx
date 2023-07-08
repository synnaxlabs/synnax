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

import { Space, Text, PIDElementSpec } from "@synnaxlabs/pluto";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { ELEMENTS } from "../elements";
import { addPIDelement } from "../store/slice";

import { CSS } from "@/css";

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
            ...ELEMENTS[type].initialProps,
          },
        })
      ),
    [dispatch, layoutKey]
  );

  return (
    <Space className={CSS.B("pid-elements")} direction="x" wrap>
      {Object.entries(ELEMENTS).map(([type, el]) => (
        <PIDElementsButton key={type} el={el} onClick={() => handleAddElement(type)} />
      ))}
    </Space>
  );
};

interface PIDElementsButtonProps
  extends PropsWithChildren,
    ComponentPropsWithoutRef<"button"> {
  el: PIDElementSpec;
}

const PIDElementsButton = ({
  children,
  el: { title, Preview },
  ...props
}: PIDElementsButtonProps): ReactElement => {
  return (
    <>
      {/* @ts-expect-error */}
      <Space
        el="button"
        className={CSS.BE("pid-elements", "button")}
        justify="center"
        align="center"
        {...props}
      >
        <Text level="p" color="var(--pluto-gray-p0)">
          {title}
        </Text>
        <Preview />
      </Space>
    </>
  );
};
