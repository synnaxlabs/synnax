// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";

import { Keys } from "@/components/keys/Keys";
import { Results } from "@/components/search/Results";

const TRIGGER: Triggers.Trigger = ["Control", "K"];

export interface SearchProps {
  initialVisible?: boolean;
}

/**
 * Renders the search button and dialog. The page renders it closed, which also puts
 * the results' styles in the page, and the search island mounts it open.
 */
export const Search = ({ initialVisible }: SearchProps): ReactElement => (
  <Triggers.Provider>
    <Dialog.Frame
      variant="modal"
      className="search-box"
      initialVisible={initialVisible}
    >
      <Dialog.Trigger
        variant="outlined"
        justify="center"
        size="medium"
        textColor={8}
        trigger={TRIGGER}
      >
        <Icon.Search />
        Search
        <Keys trigger={TRIGGER} className="search-box__keys" />
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered={false}
        pack
        rounded={1}
        className="search-results__content"
      >
        <Results />
      </Dialog.Dialog>
    </Dialog.Frame>
  </Triggers.Provider>
);
