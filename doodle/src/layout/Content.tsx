// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo, type ReactElement } from "react";

import { useOptionalRenderer } from "@/layout/context";
import { useRemover } from "@/layout/hooks";
import { useSelect, useSelectFocused } from "@/layout/selectors";

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
  ({ layoutKey, forceHidden }: ContentProps): ReactElement | null => {
    const layout = useSelect(layoutKey);
    const [, focused] = useSelectFocused();
    const handleClose = useRemover(layoutKey);
    const type = layout?.type ?? "";
    const Renderer = useOptionalRenderer(type);
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
