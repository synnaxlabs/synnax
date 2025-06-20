// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useEffect,
} from "react";
import { type z } from "zod/v4";

import { Aether } from "@/aether";
import { useMemoDeepEqualProps } from "@/memo";
import { control } from "@/telem/control/aether";

export interface ControllerProps
  extends Omit<z.input<typeof control.controllerStateZ>, "needsControlOf">,
    PropsWithChildren {
  onStatusChange?: (status: control.Status) => void;
  name: string;
}

export interface ContextValue {
  needsControlOf: channel.Keys;
}

const Context = createContext<ContextValue>({ needsControlOf: [] });

export const useContext = () => use(Context);

export const Controller = ({
  children,
  onStatusChange,
  ...props
}: ControllerProps): ReactElement => {
  const memoProps = useMemoDeepEqualProps(props);
  const [{ path }, { status, needsControlOf }, setState] = Aether.use({
    type: control.Controller.TYPE,
    schema: control.controllerStateZ,
    initialState: memoProps,
  });
  useEffect(() => {
    if (status != null) onStatusChange?.(status);
  }, [status, onStatusChange]);
  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);
  useEffect(() => () => onStatusChange?.("released"), []);

  return (
    <Context value={{ needsControlOf }}>
      <Aether.Composite path={path}>{children}</Aether.Composite>;
    </Context>
  );
};
