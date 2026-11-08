// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource, type Synnax as Client } from "@synnaxlabs/client";
import { type Menu } from "@synnaxlabs/lyra/menu";
import { type Status } from "@synnaxlabs/lyra/status";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { unique } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { useStore } from "react-redux";

import { ContextMenu } from "@/platform/context-menu";
import { Errors } from "@/platform/errors";
import { Panel } from "@/platform/panel";
import { type Item } from "@/platform/tree/item";
import { isMoreKey, type Store } from "@/platform/tree/store";
import {
  type ContextMenu as ItemContextMenu,
  type ContextMenuProps,
  type RootContextMenuProps,
  type TreeState,
} from "@/platform/tree/types";
import { useResources } from "@/platform/tree/useResource";
import { type Action, type State } from "@/session/store";

/** A menu that holds only the reload of the Console. */
export const FallbackContextMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);

interface SelectionContextMenuProps extends ContextMenuProps {
  menu: FC<ContextMenuProps>;
}

// A resource deleted while its menu is open stays in the selection. A menu reads every
// selected resource as present.
const SelectionContextMenu = ({ menu: M, ...props }: SelectionContextMenuProps) => {
  const resources = useResources(props.state, props.selection.ids);
  if (resources.length < props.selection.ids.length) return <FallbackContextMenu />;
  return <M {...props} />;
};

export interface UseMenuProps {
  client: Client;
  store: Store;
  root: resource.ID;
  selected: string[];
  state: TreeState;
  /** The menu for a right-click on empty space. */
  rootContextMenu: FC<RootContextMenuProps>;
  /** The menu for a selection of more than one resource type. */
  mixedContextMenu: ItemContextMenu;
  resolveItem: (type: resource.Type) => Item;
  handleError: Status.ErrorHandler;
}

/** Returns the context menu of a tree: for the selection, or for empty space. */
export const useMenu = ({
  client,
  store,
  root,
  selected,
  state,
  rootContextMenu: RootContextMenu,
  mixedContextMenu,
  resolveItem,
  handleError,
}: UseMenuProps): ((props: Menu.ContextMenuMenuProps) => ReactElement) => {
  const openTab = Panel.useOpenTab();
  const session = useStore<State, Action>();

  const render = ({ keys: targets }: Menu.ContextMenuMenuProps): ReactElement => {
    // The node that loads more names no resource, so it opens the root menu.
    const clicked = targets.filter((key) => !isMoreKey(key));
    const chosen = selected.filter((key) => !isMoreKey(key));
    if (clicked.length === 0) return <RootContextMenu root={root} state={state} />;
    const rightClickedButNotSelected = clicked.find((v) => !chosen.includes(v));
    // A right-click outside the selection shows a menu for that item alone. Otherwise
    // the selection state gives the keys, since the virtual tree can lack the DOM
    // elements.
    const keys =
      rightClickedButNotSelected != null ? [rightClickedButNotSelected] : chosen;
    const ids = keys.map((key) => resource.parseID(key));
    // A selected node's own children do not count, since an action on the node carries
    // them along. Flat search results show both at the same depth.
    const contained = store.getContained(keys);
    const top = Base.getAllNodesOfMinDepth(
      Base.filterShape(state.shape, (key) => keys.includes(key) && !contained.has(key)),
    );
    const parents = unique.by(
      top.map((key) => store.getAncestors(key)[0]),
      (id) => resource.idToString(id),
    );
    const props: ContextMenuProps = {
      selection: {
        rootID: root,
        parentID: parents.length === 1 ? parents[0] : null,
        ids,
        top: top.map((key) => resource.parseID(key)),
      },
      state,
      client,
      store: session,
      openTab,
      handleError,
    };
    const allSameType = ids.every((id) => id.type === ids[0].type);
    const M = allSameType ? resolveItem(ids[0].type).ContextMenu : mixedContextMenu;
    if (M == null) return <FallbackContextMenu />;
    return <SelectionContextMenu menu={M} {...props} />;
  };

  return (props) => (
    <Errors.SuspenseBoundary loading={null}>{render(props)}</Errors.SuspenseBoundary>
  );
};
