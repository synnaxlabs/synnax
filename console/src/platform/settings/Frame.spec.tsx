// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Settings } from "@/platform/settings";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

const createSection = (name: string, visible = true): Settings.Section => ({
  key: name.toLowerCase(),
  name,
  icon: <Icon.Settings />,
  Content: () => <p>{`${name} page`}</p>,
  useVisible: () => visible,
});

describe("Settings.Frame", () => {
  it("should show the page of the section the user selects", async () => {
    const groups: Settings.Group[] = [
      {
        key: "first",
        name: "First",
        sections: [createSection("Alpha"), createSection("Beta")],
      },
    ];
    await renderWithConsole(<Settings.Frame groups={groups} />);
    expect(screen.getByText("Alpha page")).toBeTruthy();
    fireEvent.click(screen.getByRole("menuitem", { name: "Beta" }));
    expect(screen.getByText("Beta page")).toBeTruthy();
    expect(screen.queryByText("Alpha page")).toBeNull();
  });

  it("should hide the sections the user cannot use and the groups left empty", async () => {
    const groups: Settings.Group[] = [
      {
        key: "first",
        name: "First",
        sections: [createSection("Hidden", false), createSection("Shown")],
      },
      { key: "second", name: "Second", sections: [createSection("Gone", false)] },
    ];
    await renderWithConsole(<Settings.Frame groups={groups} />);
    expect(screen.getByText("Shown page")).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "Hidden" })).toBeNull();
    expect(screen.queryByRole("menuitem", { name: "Gone" })).toBeNull();
    expect(screen.queryByText("Second")).toBeNull();
  });

  it("should show the section the window showed last when it opens again", async () => {
    const groups: Settings.Group[] = [
      {
        key: "first",
        name: "First",
        sections: [createSection("Alpha"), createSection("Beta")],
      },
    ];
    const { store, unmount } = await renderWithConsole(
      <Settings.Frame groups={groups} />,
    );
    fireEvent.click(screen.getByRole("menuitem", { name: "Beta" }));
    unmount();
    await renderWithConsole(<Settings.Frame groups={groups} />, { store });
    expect(screen.getByText("Beta page")).toBeTruthy();
  });

  it("should show the first section when the user cannot use the last one", async () => {
    const groups: Settings.Group[] = [
      {
        key: "first",
        name: "First",
        sections: [createSection("Shown"), createSection("Hidden", false)],
      },
    ];
    const { store, unmount } = await renderWithConsole(<p />);
    store.dispatch(Session.Nav.selectSettingsSection({ section: "hidden" }));
    unmount();
    await renderWithConsole(<Settings.Frame groups={groups} />, { store });
    expect(screen.getByText("Shown page")).toBeTruthy();
  });

  it("should drop the inner padding of a flush card", async () => {
    await renderWithConsole(
      <>
        <Settings.Card title="Padded">padded</Settings.Card>
        <Settings.Card title="Flush" flush>
          flush
        </Settings.Card>
      </>,
    );
    const FLUSH = "console-settings__card--flush";
    expect(screen.getByText("padded").classList.contains(FLUSH)).toBe(false);
    expect(screen.getByText("flush").classList.contains(FLUSH)).toBe(true);
  });
});
