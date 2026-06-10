// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { memo, type ReactElement, useMemo } from "react";
import { useDispatch } from "react-redux";

import { useRenderer } from "@/layout/context";
import { useSelect, useSelectFocused, useSelectType } from "@/layout/selectors";
import { rename, setArgs } from "@/layout/slice";
import { useRemover } from "@/layout/useRemover";
import { type RendererView, RendererViewProvider } from "@/layout/view";

/** LayoutContentProps are the props for the LayoutContent component. */
export interface ContentProps {
  layoutKey: string;
  forceHidden?: boolean;
}

/**
 * LayoutContent renders a layout given its key.
 *
 * @param props - The props for the component.
 * @param props.layoutKey - The key of the layout to render. The key must exist in the store,
 * and a renderer for the layout type must be registered in the LayoutContext.
 */
export const Content = memo(
  ({ layoutKey, forceHidden }: ContentProps): ReactElement => {
    const dispatch = useDispatch();
    const layout = useSelect(layoutKey);
    const type = useSelectType(layoutKey) ?? "";
    const handleClose = useRemover(layoutKey);
    const Renderer = useRenderer(type);
    const { focused } = useSelectFocused();
    const isFocused = focused === layoutKey;
    let visible = focused == null || isFocused;
    if (forceHidden) visible = false;
    // The layout slice is this host's backing store for the renderer's name and
    // args, served through the same RendererView channel panel tabs use so
    // arg-driven renderers read one interface regardless of host.
    const rendererView = useMemo<RendererView>(
      () => ({
        key: layoutKey,
        name: layout?.name ?? "",
        args: (layout?.args as record.Unknown) ?? {},
        update: (patch) => {
          if (patch.name != null)
            dispatch(rename({ key: layoutKey, name: patch.name }));
          if (patch.args != null)
            dispatch(setArgs({ key: layoutKey, args: patch.args }));
        },
      }),
      [layoutKey, layout?.name, layout?.args, dispatch],
    );
    return (
      <Errors.SuspenseBoundary>
        <RendererViewProvider value={rendererView}>
          <Renderer
            key={layoutKey}
            layoutKey={layoutKey}
            onClose={handleClose}
            visible={visible}
            focused={isFocused}
            active={visible}
          />
        </RendererViewProvider>
      </Errors.SuspenseBoundary>
    );
  },
);
Content.displayName = "LayoutContent";
