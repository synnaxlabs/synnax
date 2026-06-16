// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { Button, type Icon, Panel as Base } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { type Selectable } from "@/selector/Selector";

export interface ItemProps extends Omit<Button.ButtonProps, "children"> {
  title: string;
  icon: Icon.ReactElement;
}

export const Item = ({ title, icon, ...rest }: ItemProps) => (
  <Button.Button variant="outlined" {...rest}>
    {icon}
    {title}
  </Button.Button>
);

export interface SimpleItemProps {
  title: string;
  icon: Icon.ReactElement;
  layout: Layout.BaseState;
  useVisible?: () => boolean;
}

export const createSimpleItem = ({
  title,
  icon,
  layout,
  useVisible,
}: SimpleItemProps): Selectable => {
  const C: Selectable = ({ tabKey }) => {
    const visible = useVisible?.() ?? true;
    const place = Layout.usePlacer();
    const panelKey = Layout.useSelectActivePanelKey();
    const { dispatch } = Base.useDispatch();
    // In a panel the item replaces its tab with the chosen view (the layout's
    // type selects the renderer, its args/name seed the view); otherwise it
    // places the layout as before (e.g. a window form).
    const handleClick = useCallback(() => {
      if (tabKey != null && panelKey != null)
        dispatch({
          key: panelKey,
          actions: [
            panel.setTabContent({
              key: tabKey,
              type: layout.type,
              args: { ...(layout.args as record.Unknown), name: layout.name },
            }),
          ],
        });
      else place(layout);
    }, [tabKey, panelKey, dispatch, place]);
    if (!visible) return null;
    return <Item title={title} icon={icon} onClick={handleClick} />;
  };
  C.type = layout.type;
  C.useVisible = useVisible;
  return C;
};
