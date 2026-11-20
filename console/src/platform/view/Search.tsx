// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { plural } from "pluralize";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useContext, useSearchContext } from "@/platform/view/context";

/**
 * The search field of a frame, which holds the term. It saves nothing, so every reader
 * gets it. Esc or the clear button ends the search.
 */
export const Search = (): ReactElement => {
  const { resourceType } = useContext("View.Search");
  const { term, setTerm } = useSearchContext("View.Search");
  return (
    <div
      className={CSS.cls(
        CSS.BE("view", "search"),
        term !== "" && CSS.BEM("view", "search", "filled"),
      )}
    >
      <Input.Text
        size="small"
        level="h5"
        color={9}
        startContent={startContent}
        endContent={
          term === "" ? undefined : (
            <Button.Button
              variant="text"
              size="small"
              square
              textColor={9}
              tooltip="Clear search"
              tooltipLocation="bottom"
              onClick={() => setTerm("")}
            >
              <Icon.Close />
            </Button.Button>
          )
        }
        variant="text"
        grow
        value={term}
        onChange={setTerm}
        onKeyDown={(e) => {
          if (e.key === "Escape") setTerm("");
        }}
        placeholder={`Search ${plural(resourceType)}...`}
      />
    </div>
  );
};

const startContent = <Icon.Search color={9} />;
