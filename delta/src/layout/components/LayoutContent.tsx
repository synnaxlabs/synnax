// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo } from "react";

import { useDispatch } from "react-redux";

import { useLayoutRenderer } from "../context";
import { useLayoutRemover } from "../hooks";
import { useSelectRequiredLayout } from "../store";

/** LayoutContentProps are the props for the LayoutContent component. */
export interface LayoutContentProps {
  layoutKey: string;
}

/**
 * LayoutContent renders a layout given its key.
 *
 * @param props - The props for the comoponent.
 * @param props.layoutKey - The key of the layout to render. The key must exist in the store,
 * and a renderer for the layout type must be registered in the LayoutContext.
 */
export const LayoutContent = memo(
  ({ layoutKey }: LayoutContentProps): JSX.Element | null => {
    const p = useSelectRequiredLayout(layoutKey);

    const _handleClose = useLayoutRemover(layoutKey);
    const dispatch = useDispatch();

    const renderer = useLayoutRenderer(p.type);

    const Renderer = "Renderer" in renderer ? renderer.Renderer : renderer;

    const handleClose = (): void => {
      if ("onClose" in renderer) renderer.onClose?.({ layoutKey, dispatch });
      _handleClose();
    };

    if (Renderer == null) throw new Error(`layout renderer ${p.type} not found`);
    return <Renderer layoutKey={layoutKey} onClose={handleClose} />;
  }
);
LayoutContent.displayName = "LayoutContent";
