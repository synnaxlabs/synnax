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
import { useRemover } from "@/layout/hooks";
import { useSelectRequired } from "@/layout/selectors";

/** LayoutContentProps are the props for the LayoutContent component. */
export interface ContentProps {
  layoutKey: string;
}

/**
 * LayoutContent renders a layout given its key.
 *
 * @param props - The props for the comoponent.
 * @param props.layoutKey - The key of the layout to render. The key must exist in the store,
 * and a renderer for the layout type must be registered in the LayoutContext.
 */
export const Content = memo(({ layoutKey }: ContentProps): ReactElement | null => {
  const p = useSelectRequired(layoutKey);
  const handleClose = useRemover(layoutKey);
  const Renderer = useLayoutRenderer(p.type);
  if (Renderer == null) throw new Error(`layout renderer ${p.type} not found`);
  return <Renderer key={layoutKey} layoutKey={layoutKey} onClose={handleClose} />;
});
Content.displayName = "LayoutContent";
