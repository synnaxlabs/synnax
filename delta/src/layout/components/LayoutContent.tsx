// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, memo } from "react";

import { useLayoutRenderer } from "@/layout/context";
import { useLayoutRemover } from "@/layout/hooks";
import { useSelectRequiredLayout } from "@/layout/store";

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
  ({ layoutKey }: LayoutContentProps): ReactElement | null => {
    const p = useSelectRequiredLayout(layoutKey);
    const handleClose = useLayoutRemover(layoutKey);
    const Renderer = useLayoutRenderer(p.type);
    if (Renderer == null) throw new Error(`layout renderer ${p.type} not found`);
    return <Renderer layoutKey={layoutKey} onClose={handleClose} />;
  }
);
LayoutContent.displayName = "LayoutContent";
