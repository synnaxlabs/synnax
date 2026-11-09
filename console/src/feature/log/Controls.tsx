// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { Log as Base } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { memo, type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

// Keeps a press or a copy in the search box from reaching the log around it.
const stopPropagation = (e: React.SyntheticEvent): void => e.stopPropagation();

// A button that opens a box to find and filter the lines of a log.
const Search = (): ReactElement => {
  const key = Base.useKey();
  const dispatch = Session.useDispatch();
  const search = Session.Log.useSelectSearch();
  // A stored search shows, so the lines it hides are never a surprise.
  const [open, setOpen] = useState(search.text !== "");
  const set = (change: Partial<Session.Log.SearchState>): void => {
    dispatch(Session.Log.setSearch({ key, ...change }));
  };
  const close = (): void => {
    set({ text: "" });
    setOpen(false);
  };
  if (!open)
    return (
      <Button.Button
        size="small"
        onClick={() => setOpen(true)}
        tooltip="Search"
        tooltipLocation={location.BOTTOM_LEFT}
        aria-label="Search"
      >
        <Icon.Search />
      </Button.Button>
    );
  const invalid = Base.search.invalid(search);
  return (
    <Flex.Box
      y
      gap="tiny"
      className={CSS.cls(
        CSS.BE("log", "search"),
        search.text !== "" && Vis.CONTROLS_PINNED_CLASS,
      )}
      onMouseDown={stopPropagation}
      onCopy={stopPropagation}
    >
      <Flex.Box x gap="tiny" align="center">
        <Input.Text
          value={search.text}
          onChange={(text) => set({ text })}
          size="small"
          autoFocus
          placeholder="Search"
          aria-label="Search text"
          status={invalid ? "error" : undefined}
          onKeyDown={(e) => {
            if (e.key === "Escape") close();
          }}
        />
        <Button.Toggle
          value={search.mode === "regex"}
          onChange={(regex) => set({ mode: regex ? "regex" : "text" })}
          size="small"
          tooltip="Use regex"
          aria-label="Use regex"
        >
          .*
        </Button.Toggle>
        <Button.Toggle
          value={search.filtered}
          onChange={(filtered) => set({ filtered })}
          size="small"
          tooltip="Show only matching lines"
          aria-label="Show only matching lines"
        >
          <Icon.Filter />
        </Button.Toggle>
        <Button.Button
          size="small"
          variant="text"
          onClick={close}
          tooltip={
            <Text.Text level="small" color={11}>
              Close
              <Triggers.Text trigger={Triggers.ESCAPE} level="small" />
            </Text.Text>
          }
          aria-label="Close search"
        >
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
      {invalid && <Input.HelpText variant="error">Invalid regex</Input.HelpText>}
    </Flex.Box>
  );
};

export const Controls = memo((): ReactElement => {
  const key = Base.useKey();
  const hold = Session.Log.useSelectHold();
  const dispatch = Session.useDispatch();
  const handleHoldChange = (hold: boolean): void => {
    dispatch(Session.Log.setHold({ key, hold }));
  };
  return (
    <Vis.Controls>
      <Search />
      <Button.Toggle
        value={hold}
        onChange={handleHoldChange}
        size="small"
        aria-label="Pause scrolling"
        className={CSS.cls(hold && Vis.CONTROLS_PINNED_CLASS)}
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={
          <Text.Text level="small" color={11}>
            {`${hold ? "Resume" : "Pause"} scrolling`}
            <Triggers.Text trigger={Base.PAUSE_TRIGGER} level="small" />
          </Text.Text>
        }
      >
        {hold ? <Icon.Play /> : <Icon.Pause />}
      </Button.Toggle>
    </Vis.Controls>
  );
});
Controls.displayName = "Controls";
