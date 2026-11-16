// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, type FC, type PropsWithChildren } from "react";

import { getBySelector, getIconButton } from "@/testutil";

/** Nests a test wrapper's children in a panel tab, which a View.Frame renders in. */
export const inTab = (
  Wrapper: FC<PropsWithChildren>,
  tabKey: panel.TabKey = uuid.create(),
): FC<PropsWithChildren> => {
  const Tab: FC<PropsWithChildren> = ({ children }) =>
    createElement(
      Wrapper,
      null,
      createElement(Panel.TabProvider, { tabKey }, children),
    );
  return Tab;
};

/**
 * Waits for the icon button with the given icon name in the mounted View.Toolbar, and
 * returns it.
 */
export const findToolbarIconButton = async (icon: string): Promise<HTMLButtonElement> =>
  await waitFor(() =>
    getIconButton(getBySelector(document.body, ".console-view__toolbar"), icon),
  );

/**
 * Opens the filter menu of the mounted editable view, then the submenu of the property
 * with the name, and returns the picker it opens.
 */
export const openFilterProperty = async (name: string): Promise<HTMLElement> => {
  fireEvent.click(await waitFor(() => getIconButton(document.body, "filter")));
  fireEvent.click(await screen.findByRole("menuitem", { name }));
  return await screen.findByRole("dialog", { name });
};
