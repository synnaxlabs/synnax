// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo, type ReactElement } from "react";

import { useRenderer } from "@/layout/RendererProvider";
import { useSelectFocused, useSelectType } from "@/layout/selectors";
import { useRemover } from "@/layout/useRemover";

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
    const type = useSelectType(layoutKey) ?? "";
    const handleClose = useRemover(layoutKey);
    const Renderer = useRenderer(type);
    const { focused } = useSelectFocused();
    if (Renderer == null) throw new Error(`layout renderer ${type} not found`);
    const isFocused = focused === layoutKey;
    let visible = focused == null || isFocused;
    if (forceHidden) visible = false;
    return (
      <Renderer
        key={layoutKey}
        layoutKey={layoutKey}
        onClose={handleClose}
        visible={visible}
        focused={isFocused}
      />
    );
  },
);
Content.displayName = "LayoutContent";
