// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, type rack, type Synnax as Client } from "@synnaxlabs/client";
import { CERTIFICATE, createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
import { deep, TimeStamp, uuid } from "@synnaxlabs/x";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, assert, describe, expect, it, type Mock, vi } from "vitest";
import { z } from "zod";

import { Certificate } from "@/platform/certificate";
import {
  createFileRack,
  createListing,
  serveListings,
} from "@/platform/file-browser/testutil";
import {
  createFile,
  fakeFileEntry,
  FileDragSource,
  fireFileDrop,
  startFileDrag,
} from "@/platform/fs/testutil";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Secret } from "@/platform/secret";
import {
  awaitGranted,
  captureBrowserDownloads,
  CaptureStatuses,
  createConsoleWrapper,
  createTestClientWithGrants,
  fakePickedFile,
  findDialogTriggerByText,
  getIconButton,
  interceptFilePicker,
  queryIconButton,
  readBlob,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const SECRETS_DIR = "/etc/synnax/secrets";

const created: certificate.Key[] = [];

afterEach(async () => {
  cleanup();
  vi.restoreAllMocks();
  await client.certificates.delete(created.splice(0));
});

const DER = Uint8Array.from(
  atob(CERTIFICATE.pem.replace(/-----(BEGIN|END) CERTIFICATE-----|\s/g, "")),
  (c) => c.charCodeAt(0),
);

const createStored = async (): Promise<certificate.StoredCertificate> => {
  const c = await client.certificates.set({
    name: uniqueName("certificate"),
    source: "stored",
    pem: CERTIFICATE.pem,
  });
  created.push(c.key);
  assert(c.source === "stored");
  return c;
};

const schema = z.object({ certificate: z.string() });

const path = deep.path<z.infer<typeof schema>>();

interface FixtureProps {
  value: string;
  defaultName: string;
  rack: rack.Key;
  onChange: (key: string) => void;
  mode?: Form.Mode;
  allowNone?: boolean;
}

const Fixture = ({
  value,
  defaultName,
  rack,
  onChange,
  mode,
  allowNone,
}: FixtureProps): ReactElement => {
  const form = Form.use({
    schema,
    mode,
    values: { certificate: value },
    onChange: ({ values }) => onChange(values.certificate),
  });
  return (
    <Form.Form<typeof schema> {...form}>
      <Certificate.Field
        path={path.certificate}
        defaultName={defaultName}
        rack={rack}
        allowNone={allowNone}
      />
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

interface RenderedField {
  defaultName: string;
  onChange: Mock<(key: string) => void>;
  statuses: Status.NotificationSpec[];
  /** Renders the field again with another rack, as when the device's rack changes. */
  setRack: (rack: rack.Key) => void;
}

interface RenderOptions {
  as?: Client;
  mode?: Form.Mode;
  rack?: rack.Key;
  allowNone?: boolean;
}

const renderField = async (
  value: string = "",
  { as = client, mode, rack = 0, allowNone }: RenderOptions = {},
): Promise<RenderedField> => {
  const defaultName = uniqueName("default");
  const onChange = vi.fn<(key: string) => void>();
  const statuses: Status.NotificationSpec[] = [];
  const { wrapper } = await createConsoleWrapper({ client: as });
  const content = (fieldRack: rack.Key): ReactElement => (
    <Haul.Provider>
      <FileDragSource />
      <Fixture
        value={value}
        defaultName={defaultName}
        rack={fieldRack}
        onChange={onChange}
        mode={mode}
        allowNone={allowNone}
      />
      <Modals.Stack />
      <CaptureStatuses
        onStatuses={(next) => statuses.splice(0, statuses.length, ...next)}
      />
    </Haul.Provider>
  );
  const { rerender } = render(content(rack), { wrapper });
  return {
    defaultName,
    onChange,
    statuses,
    setRack: (next) => rerender(content(next)),
  };
};

const open = async (shown: string): Promise<void> => {
  fireEvent.click(await findDialogTriggerByText(shown));
};

// The list pages, so a search finds a certificate however many the Core holds.
const findOption = async (c: certificate.Certificate): Promise<HTMLElement> => {
  fireEvent.change(await screen.findByPlaceholderText("Search certificates..."), {
    target: { value: c.name },
  });
  return await screen.findByRole("option", { name: new RegExp(c.name) });
};

const openCreate = async (): Promise<void> => {
  await open("Select a certificate");
  fireEvent.click(await screen.findByText("New certificate"));
  await screen.findByLabelText("Certificate name");
};

// A required field's label ends in an asterisk.
const getPEMInput = (label: string): HTMLElement =>
  screen.getByRole("textbox", { name: new RegExp(`^${label}( \\*)?$`) });

const typeInto = (label: string, value: string): void => {
  fireEvent.change(getPEMInput(label), { target: { value } });
};

// Saves the create dialog and returns the stored certificate that the field selected.
const saveCreated = async (
  onChange: Mock<(key: string) => void>,
): Promise<certificate.StoredCertificate> => {
  fireEvent.click(findButton("Create"));
  const key = await waitFor(() => {
    const [last] = onChange.mock.lastCall ?? [];
    if (last == null) throw new Error("the field did not change");
    return last;
  });
  created.push(key);
  const c = await client.certificates.retrieve(key);
  assert(c.source === "stored");
  return c;
};

// Pairs each term of the details card with its value.
const readDetails = (): Record<string, string> => {
  const terms = screen.getAllByRole("term");
  const values = screen.getAllByRole("definition");
  return Object.fromEntries(
    terms.map((term, i) => [term.textContent, values[i].textContent.trim()]),
  );
};

const openDownload = async (): Promise<void> => {
  await screen.findByText("SHA-256");
  fireEvent.click(getIconButton(document.body, "download"));
};

const fingerprintOf = (c: certificate.Certificate): string => {
  assert(c.details != null);
  return c.details.sha256.toUpperCase().match(/.{2}/g)?.join(":") ?? "";
};

describe("Certificate.Field", () => {
  it("lists each certificate with its validity", async () => {
    const c = await createStored();
    await renderField();
    await open("Select a certificate");
    const option = await findOption(c);
    within(option).getByText(
      Secret.Validity.describeCertificate(c, TimeStamp.now()).text,
    );
  });

  it("shows no prompt to pick a certificate in preview", async () => {
    await renderField("", { mode: "preview" });
    await screen.findByText("None");
    expect(screen.queryByText("Select a certificate")).toBeNull();
  });

  it("shows that the certificate is not found for a key that names none", async () => {
    await renderField(uuid.create());
    await findDialogTriggerByText("Certificate not found");
  });

  it("shows that the certificate is unavailable to a user who cannot retrieve it", async () => {
    const c = await createStored();
    await renderField(c.key, { as: await createTestClientWithGrants(client) });
    await findDialogTriggerByText("Unavailable certificate");
    expect(screen.queryByText(c.name)).toBeNull();
  });

  it("sets the value to the key of the picked certificate and shows its details", async () => {
    const c = await createStored();
    const { onChange } = await renderField();
    await open("Select a certificate");
    fireEvent.click(await findOption(c));
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith(c.key));
    await findDialogTriggerByText(c.name);
    await screen.findByText("Subject");
    expect(readDetails()).toEqual({
      Subject: c.details.subject,
      Issuer: c.details.issuer,
      Validity: Secret.Validity.describeCertificate(c, TimeStamp.now()).text,
      "SHA-256": fingerprintOf(c),
    });
  });

  it("clears the certificate on Delete when it allows none", async () => {
    const c = await createStored();
    const { onChange } = await renderField(c.key, { allowNone: true });
    fireEvent.keyDown(await findDialogTriggerByText(c.name), { key: "Delete" });
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith(""));
  });

  it("offers no clear when it does not allow none", async () => {
    const c = await createStored();
    await renderField(c.key);
    fireEvent.keyDown(await findDialogTriggerByText(c.name), { key: "Delete" });
    expect(screen.queryByLabelText(`Clear ${c.name}`)).toBeNull();
    expect(await findDialogTriggerByText(c.name)).toBeTruthy();
  });

  it("downloads the chain as PEM", async () => {
    const downloads = captureBrowserDownloads();
    const c = await createStored();
    await renderField(c.key);
    await openDownload();
    fireEvent.click(await screen.findByText("Download as PEM"));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${c.name}.pem`);
    expect(await readBlob(downloads.blobs[0])).toBe(c.pem);
  });

  it("downloads the leaf as DER", async () => {
    const downloads = captureBrowserDownloads();
    const c = await createStored();
    await renderField(c.key);
    await openDownload();
    fireEvent.click(await screen.findByText("Download as DER"));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${c.name}.der`);
    const bytes = new Uint8Array(await downloads.blobs[0].arrayBuffer());
    expect(bytes).toEqual(DER);
  });

  it("shows the path of a file certificate and offers no download", async () => {
    const r = await client.racks.create({ name: uniqueName("rack") });
    const c = await client.certificates.set({
      name: uniqueName("file"),
      source: "file",
      certificatePath: "/etc/synnax/client.crt",
      rack: r.key,
    });
    created.push(c.key);
    await renderField(c.key, { rack: r.key });
    await screen.findByRole("term");
    expect(readDetails()).toEqual({ Path: "/etc/synnax/client.crt" });
    expect(queryIconButton(document.body, "download")).toBeNull();
  });

  it("offers no new certificate to a user who cannot create certificates", async () => {
    const c = await createStored();
    const reader = await createTestClientWithGrants(client, {
      retrieve: [certificate.TYPE_RESOURCE_ID],
    });
    await awaitGranted(reader, certificate.TYPE_RESOURCE_ID, "retrieve");
    await renderField("", { as: reader });
    await open("Select a certificate");
    await findOption(c);
    expect(screen.queryByText("New certificate")).toBeNull();
  });

  it("opens the create dialog with the default name and focuses the certificate", async () => {
    const { defaultName } = await renderField();
    await openCreate();
    expect(screen.getByLabelText("Certificate name")).toHaveProperty(
      "value",
      defaultName,
    );
    await waitFor(() =>
      expect(document.activeElement).toBe(getPEMInput("Certificate")),
    );
  });

  it("creates a certificate from pasted PEM with its key and selects it", async () => {
    const { defaultName, onChange } = await renderField();
    await openCreate();
    typeInto("Certificate", CERTIFICATE.pem);
    typeInto("Private key", CERTIFICATE.privateKeyPem);
    const c = await saveCreated(onChange);
    expect(c.name).toBe(defaultName);
    expect(c.privateKey).toBeDefined();
    await findDialogTriggerByText(defaultName);
  });

  it("creates a certificate with no key from a dropped DER file", async () => {
    const { onChange } = await renderField();
    await openCreate();
    startFileDrag();
    fireFileDrop(getPEMInput("Certificate"), [
      fakeFileEntry(createFile("server.der", DER)),
    ]);
    await waitFor(() =>
      expect(getPEMInput("Certificate")).toHaveProperty(
        "value",
        expect.stringContaining("-----BEGIN CERTIFICATE-----"),
      ),
    );
    const c = await saveCreated(onChange);
    expect(c.pem).toBe(CERTIFICATE.pem);
    expect(c.privateKey).toBeUndefined();
  });

  it("reads a dropped PEM file as it is", async () => {
    await renderField();
    await openCreate();
    startFileDrag();
    fireFileDrop(getPEMInput("Certificate"), [
      fakeFileEntry(createFile("server.crt", CERTIFICATE.pem)),
    ]);
    await waitFor(() =>
      expect(getPEMInput("Certificate")).toHaveProperty("value", CERTIFICATE.pem),
    );
  });

  it("refuses an empty certificate file", async () => {
    const { statuses } = await renderField();
    await openCreate();
    startFileDrag();
    fireFileDrop(getPEMInput("Certificate"), [
      fakeFileEntry(createFile("empty.crt", "")),
    ]);
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          message: "Failed to read file",
          description: "empty.crt is empty.",
        }),
      ),
    );
    expect(getPEMInput("Certificate")).toHaveProperty("value", "");
  });

  it("reads the key from a selected file", async () => {
    const picker = interceptFilePicker();
    const { onChange } = await renderField();
    await openCreate();
    typeInto("Certificate", CERTIFICATE.pem);
    const [, selectKey] = screen.getAllByText("Select file");
    fireEvent.click(selectKey);
    act(() =>
      picker.selectFiles([fakePickedFile("client.key", CERTIFICATE.privateKeyPem)]),
    );
    await waitFor(() =>
      expect(getPEMInput("Private key")).toHaveProperty(
        "value",
        CERTIFICATE.privateKeyPem,
      ),
    );
    const c = await saveCreated(onChange);
    expect(c.privateKey).toBeDefined();
  });

  it("leaves the key empty when a dropped key file is not PEM", async () => {
    const { statuses } = await renderField();
    await openCreate();
    startFileDrag();
    fireFileDrop(getPEMInput("Private key"), [
      fakeFileEntry(createFile("client.der", DER)),
    ]);
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          message: "Failed to read file",
          description: "client.der is not a PEM file.",
        }),
      ),
    );
    expect(getPEMInput("Private key")).toHaveProperty("value", "");
  });

  it("marks the certificate as required and requires it before it saves", async () => {
    const { onChange } = await renderField();
    await openCreate();
    expect(screen.getByRole("textbox", { name: "Certificate *" })).toBeDefined();
    expect(screen.getByRole("textbox", { name: "Private key" })).toBeDefined();
    fireEvent.click(findButton("Create"));
    await screen.findByText("Certificate is required");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("holds Cancel while the certificate saves", async () => {
    const { onChange } = await renderField();
    await openCreate();
    typeInto("Certificate", CERTIFICATE.pem);
    fireEvent.click(findButton("Create"));
    fireEvent.click(findButton("Cancel"));
    const key = await waitFor(() => {
      const [last] = onChange.mock.lastCall ?? [];
      if (last == null) throw new Error("the field did not change");
      return last;
    });
    created.push(key);
  });

  it("leaves the value unchanged when the create dialog is canceled", async () => {
    const c = await createStored();
    const { onChange } = await renderField(c.key);
    await open(c.name);
    fireEvent.click(await screen.findByText("New certificate"));
    fireEvent.click(findButton("Cancel"));
    await waitFor(() => expect(screen.queryByLabelText("Certificate name")).toBeNull());
    await act(async () => {});
    expect(onChange).not.toHaveBeenCalled();
    await findDialogTriggerByText(c.name);
  });

  it("shows a held file certificate of another rack as an error", async () => {
    const [own, other] = await Promise.all([
      client.racks.create({ name: uniqueName("rack") }),
      client.racks.create({ name: uniqueName("rack") }),
    ]);
    const c = await client.certificates.set({
      name: uniqueName("file"),
      source: "file",
      certificatePath: "/etc/synnax/client.crt",
      rack: own.key,
    });
    created.push(c.key);
    const { setRack } = await renderField(c.key, { rack: own.key });
    await findDialogTriggerByText(c.name);
    expect(screen.queryByText("File on another rack")).toBeNull();
    setRack(other.key);
    const shown = await screen.findByText("File on another rack");
    expect(shown.classList).toContain("lyra--status-error");
  });

  it("offers no file to a field with no rack", async () => {
    await renderField();
    await openCreate();
    await screen.findByLabelText("Certificate name");
    expect(screen.queryByText("File on rack")).toBeNull();
  });

  it("creates a file certificate with the details of the picked file", async () => {
    const stored = await createStored();
    const files = await createFileRack(client);
    await serveListings(client, files, [
      createListing({
        secretsDir: SECRETS_DIR,
        entries: [{ name: "/", path: "/", kind: "directory" }],
      }),
      createListing({
        path: "/",
        entries: [
          {
            name: "client.crt",
            path: "/client.crt",
            kind: "file",
            size: 1200,
            certificate: stored.details,
          },
        ],
      }),
      createListing({
        path: SECRETS_DIR,
        secret: true,
        entries: [
          { name: "client.key", path: `${SECRETS_DIR}/client.key`, kind: "file" },
        ],
      }),
    ]);
    const { defaultName, onChange } = await renderField("", { rack: files.rack.key });
    await openCreate();
    fireEvent.click(await screen.findByText("File on rack"));
    fireEvent.click(screen.getByRole("button", { name: "Browse" }));
    fireEvent.doubleClick(await screen.findByText("/"));
    fireEvent.doubleClick(await screen.findByText("client.crt"));
    await screen.findByText(new RegExp(`^${stored.details.subject} · Expires `));
    await open("No file");
    fireEvent.click(await screen.findByRole("option", { name: "client.key" }));
    fireEvent.click(findButton("Create"));
    const key = await waitFor(() => {
      const [last] = onChange.mock.lastCall ?? [];
      if (last == null) throw new Error("the field did not change");
      return last;
    });
    created.push(key);
    expect(await client.certificates.retrieve({ key })).toEqual(
      expect.objectContaining({
        name: defaultName,
        source: "file",
        certificatePath: "/client.crt",
        privateKeyFile: "client.key",
        details: expect.objectContaining({ sha256: stored.details.sha256 }),
      }),
    );
  });

  it("saves no details for a path typed by hand", async () => {
    const files = await createFileRack(client);
    await serveListings(client, files, [createListing({ secretsDir: SECRETS_DIR })]);
    const { onChange } = await renderField("", { rack: files.rack.key });
    await openCreate();
    fireEvent.click(await screen.findByText("File on rack"));
    fireEvent.change(screen.getByPlaceholderText("/etc/ssl/certs/client.crt"), {
      target: { value: "/etc/ssl/certs/client.crt" },
    });
    fireEvent.click(findButton("Create"));
    const key = await waitFor(() => {
      const [last] = onChange.mock.lastCall ?? [];
      if (last == null) throw new Error("the field did not change");
      return last;
    });
    created.push(key);
    const c = await client.certificates.retrieve({ key });
    assert(c.source === "file");
    expect(c.certificatePath).toBe("/etc/ssl/certs/client.crt");
    expect(c.details).toBeUndefined();
    expect(c.privateKeyFile).toBeUndefined();
  });

  it("marks the certificate file as required and requires it for a file certificate", async () => {
    const files = await createFileRack(client);
    await serveListings(client, files, [createListing({ secretsDir: SECRETS_DIR })]);
    const { onChange } = await renderField("", { rack: files.rack.key });
    await openCreate();
    fireEvent.click(await screen.findByText("File on rack"));
    await screen.findByRole("textbox", { name: "Certificate file *" });
    fireEvent.click(findButton("Create"));
    await screen.findByText("Certificate file is required");
    expect(onChange).not.toHaveBeenCalled();
  });
  it("lists stored certificates and the file certificates of its rack, not of another", async () => {
    const [own, other] = await Promise.all([
      client.racks.create({ name: uniqueName("rack") }),
      client.racks.create({ name: uniqueName("rack") }),
    ]);
    const prefix = uniqueName("rack certificate");
    const createFile = async (
      r: rack.Rack,
      name: string,
    ): Promise<certificate.Certificate> => {
      const c = await client.certificates.set({
        name: `${prefix} ${name}`,
        source: "file",
        certificatePath: "/etc/ssl/client.crt",
        rack: r.key,
      });
      created.push(c.key);
      return c;
    };
    const onRack = await createFile(own, "own");
    const offRack = await createFile(other, "other");
    const stored = await createStored();
    await renderField("", { rack: own.key });
    await open("Select a certificate");
    await findOption(stored);
    fireEvent.change(screen.getByPlaceholderText("Search certificates..."), {
      target: { value: prefix },
    });
    await screen.findByRole("option", { name: new RegExp(onRack.name) });
    expect(screen.queryByRole("option", { name: new RegExp(offRack.name) })).toBeNull();
  });
});
