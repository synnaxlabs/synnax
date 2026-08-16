// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { createResource } from "@/platform/tree/testutil";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  selectTestProject,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

describe("log/search", () => {
  it("opens the log as a tab when the search result is selected", async () => {
    const key = uuid.create();
    const name = uniqueName("log");
    const resource = createResource(log.ontologyID(key), name);
    const SearchListItem = Log.SEARCH_LIST_ITEMS.log;
    if (SearchListItem == null) throw new Error("log SearchListItem is not defined");
    const Harness = (): ReactElement => {
      const staticProps = List.useStaticData<string, ontology.Resource>({
        data: [resource],
      });
      return (
        <Select.Frame<string, ontology.Resource>
          {...staticProps}
          value={undefined}
          onChange={() => {}}
        >
          <SearchListItem key={resource.key} itemKey={resource.key} index={0} />
        </Select.Frame>
      );
    };
    const { wrapper, store } = await createConsoleWrapper({ client });
    await selectTestProject(store, client);
    render(<Harness />, { wrapper });
    fireEvent.click(await screen.findByText(name), { detail: 0 });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.key).toBe(key);
  });
});
