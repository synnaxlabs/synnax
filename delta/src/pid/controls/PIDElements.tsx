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
  PropsWithoutRef,
  ReactElement,
  useCallback,
} from "react";

import { ValuePIDElementSpec, Space, Text } from "@synnaxlabs/pluto";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { useSelectPID } from "../store/selectors";
import { addPIDelement } from "../store/slice";

import { CSS } from "@/css";

import "@/pid/controls/PIDElements.css";

export const PIDElements = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const dispatch = useDispatch();

  const handleAddElement = useCallback(
    () =>
      dispatch(
        addPIDelement({
          layoutKey,
          key: nanoid(),
          props: ValuePIDElementSpec.initialProps,
        })
      ),
    [dispatch, layoutKey]
  );

  return (
    <Space className={CSS.B("pid-elements")}>
      <PIDElementsButton title={ValuePIDElementSpec.title} onClick={handleAddElement}>
        <ValuePIDElementSpec.Preview />
      </PIDElementsButton>
    </Space>
  );
};

interface PIDElementsButtonProps
  extends PropsWithChildren,
    ComponentPropsWithoutRef<"button"> {
  title: string;
}

const PIDElementsButton = ({
  children,
  title,
  ...props
}: PIDElementsButtonProps): ReactElement => {
  return (
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
      {children}
    </Space>
  );
};
