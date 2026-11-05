// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { act, renderHook, type RenderHookResult } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";

import { type Selectable } from "@/platform/selector/types";

type Wrapper = FC<PropsWithChildren>;

/** Picks selectable the way a click on its button does. */
export const select = (selectable: Selectable, wrapper: Wrapper): void => {
  const { useOnSelect } = selectable;
  const useSelect = (): (() => void) => useOnSelect();
  const { result } = renderHook(useSelect, { wrapper });
  act(() => result.current());
};

export interface Visibility {
  visible: boolean;
  /** True once the user may retrieve the id, which proves their policies loaded. */
  loaded: boolean;
}

/**
 * Renders the useVisible hook of selectable for the user of wrapper, beside a retrieve
 * check on id. Wait for loaded before reading a false visible, or the check may still
 * be in flight.
 */
export const renderVisible = (
  selectable: Selectable,
  id: ontology.ID,
  wrapper: Wrapper,
): RenderHookResult<Visibility, unknown> => {
  const { useVisible } = selectable;
  if (useVisible == null) throw new Error(`${selectable.type} has no useVisible`);
  const useVisibility = (): Visibility => ({
    visible: useVisible(),
    loaded: Access.useRetrieveGranted(id),
  });
  return renderHook(useVisibility, { wrapper });
};
