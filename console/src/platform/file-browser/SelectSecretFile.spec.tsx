// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { afterEach, describe, expect, it, type Mock, vi } from "vitest";

import { FileBrowser } from "@/platform/file-browser";
import {
  createFileRack,
  createListing,
  serveListings,
} from "@/platform/file-browser/testutil";
import { createConsoleWrapper, findDialogTriggerByText } from "@/testutil";

const client = createTestClient();

afterEach(() => cleanup());

const SECRETS_DIR = "/etc/synnax/secrets";

interface FixtureProps {
  rack: rack.Key;
  onChange: (name: string) => void;
  allowNone: boolean;
}

const Fixture = ({ rack, onChange, allowNone }: FixtureProps): ReactElement => {
  const [value, setValue] = useState("");
  const handleChange = (next: string) => {
    setValue(next);
    onChange(next);
  };
  return (
    <FileBrowser.SelectSecretFile
      rack={rack}
      value={value}
      onChange={handleChange}
      allowNone={allowNone}
    />
  );
};

const renderSelect = async (
  listings: ReturnType<typeof createListing>[],
  allowNone = false,
): Promise<Mock<(name: string) => void>> => {
  const files = await createFileRack(client);
  await serveListings(client, files, listings);
  const onChange = vi.fn<(name: string) => void>();
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Fixture rack={files.rack.key} onChange={onChange} allowNone={allowNone} />, {
    wrapper,
  });
  return onChange;
};

describe("FileBrowser.SelectSecretFile", () => {
  it("should set the name of a file picked in the secrets folder", async () => {
    const onChange = await renderSelect([
      createListing({ secretsDir: SECRETS_DIR }),
      createListing({
        path: SECRETS_DIR,
        secret: true,
        entries: [
          { name: "keys", path: `${SECRETS_DIR}/keys`, kind: "directory" },
          { name: "token", path: `${SECRETS_DIR}/token`, kind: "file" },
        ],
      }),
    ]);
    fireEvent.click(await findDialogTriggerByText("Select a file"));
    expect(screen.queryByRole("option", { name: "No file" })).toBeNull();
    fireEvent.click(await screen.findByRole("option", { name: "token" }));
    expect(screen.queryByRole("option", { name: "keys" })).toBeNull();
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith("token"));
    await findDialogTriggerByText("token");
  });

  it("should clear an optional file with the No file option", async () => {
    const onChange = await renderSelect(
      [
        createListing({ secretsDir: SECRETS_DIR }),
        createListing({
          path: SECRETS_DIR,
          secret: true,
          entries: [{ name: "key", path: `${SECRETS_DIR}/key`, kind: "file" }],
        }),
      ],
      true,
    );
    fireEvent.click(await findDialogTriggerByText("No file"));
    fireEvent.click(await screen.findByRole("option", { name: "key" }));
    fireEvent.click(await findDialogTriggerByText("key"));
    fireEvent.click(await screen.findByRole("option", { name: "No file" }));
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith(""));
    await findDialogTriggerByText("No file");
  });

  it("should share each listing between two selects of one rack", async () => {
    const files = await createFileRack(client);
    const served = await serveListings(client, files, [
      createListing({ secretsDir: SECRETS_DIR }),
      createListing({
        path: SECRETS_DIR,
        secret: true,
        entries: [{ name: "token", path: `${SECRETS_DIR}/token`, kind: "file" }],
      }),
    ]);
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        <Fixture rack={files.rack.key} onChange={vi.fn()} allowNone={false} />
        <Fixture rack={files.rack.key} onChange={vi.fn()} allowNone={false} />
      </>,
      { wrapper },
    );
    const [first] = await screen.findAllByText("Select a file");
    fireEvent.click(first);
    await screen.findByRole("option", { name: "token" });
    expect(served.paths).toEqual(["", SECRETS_DIR]);
  });

  it("should say when the Driver has no secrets folder", async () => {
    await renderSelect([createListing()], true);
    const trigger = await findDialogTriggerByText("The Driver has no secrets folder");
    expect(trigger.getAttribute("aria-disabled")).toBe("true");
  });

  it("should show the error when the rack fails to list the folder", async () => {
    await renderSelect([createListing({ secretsDir: SECRETS_DIR })]);
    fireEvent.click(await findDialogTriggerByText("Select a file"));
    await screen.findByText(`${SECRETS_DIR} does not exist.`);
    expect(screen.getByText("Failed to list files")).toBeDefined();
  });
});
