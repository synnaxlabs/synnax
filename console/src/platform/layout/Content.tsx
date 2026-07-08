// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo, type ReactElement, useCallback } from "react";

import { Errors } from "@/platform/errors";
import { useRenderer } from "@/platform/layout/context";
import { useRemover } from "@/platform/layout/useRemover";
import { Session } from "@/session";

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
    const type = Session.Layout.useSelectType(layoutKey) ?? "";
    const removeLayout = useRemover(layoutKey);
    const handleClose = useCallback(() => removeLayout(), [removeLayout]);
    const Renderer = useRenderer(type);
    const { focused } = Session.Layout.useSelectFocused();
    const isFocused = focused === layoutKey;
    let visible = focused == null || isFocused;
    if (forceHidden) visible = false;
    return (
      <Errors.SuspenseBoundary layoutKey={layoutKey}>
        <Renderer
          key={layoutKey}
          layoutKey={layoutKey}
          onClose={handleClose}
          visible={visible}
          focused={isFocused}
        />
      </Errors.SuspenseBoundary>
    );
  },
);
Content.displayName = "LayoutContent";
