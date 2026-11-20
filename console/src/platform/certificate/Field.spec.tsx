// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, type Synnax as Client } from "@synnaxlabs/client";
import { CERTIFICATE, createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
import { deep, TimeStamp, uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
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
  createFile,
  fakeFileEntry,
  FileDragSource,
  fireFileDrop,
  startFileDrag,
} from "@/platform/fs/testutil";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { findOption } from "@/platform/picker/testutil";
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
} from "@/testutil";

const client = createTestClient();

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
  onChange: (key: string) => void;
  mode?: Form.Mode;
}

const Fixture = ({
  value,
  defaultName,
  onChange,
  mode,
}: FixtureProps): ReactElement => {
  const form = Form.use({
    schema,
    mode,
    values: { certificate: value },
    onChange: ({ values }) => onChange(values.certificate),
  });
  return (
    <Form.Form<typeof schema> {...form}>
      <Certificate.Field path={path.certificate} defaultName={defaultName} />
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

interface RenderedField {
  defaultName: string;
  onChange: Mock<(key: string) => void>;
  statuses: Status.NotificationSpec[];
}

const renderField = async (
  value: string = "",
  as: Client = client,
  mode?: Form.Mode,
): Promise<RenderedField> => {
  const defaultName = uniqueName("default");
  const onChange = vi.fn<(key: string) => void>();
  const statuses: Status.NotificationSpec[] = [];
  const { wrapper } = await createConsoleWrapper({ client: as });
  render(
    <Haul.Provider>
      <FileDragSource />
      <Fixture
        value={value}
        defaultName={defaultName}
        onChange={onChange}
        mode={mode}
      />
      <Modals.Stack />
      <CaptureStatuses
        onStatuses={(next) => statuses.splice(0, statuses.length, ...next)}
      />
    </Haul.Provider>,
    { wrapper },
  );
  return { defaultName, onChange, statuses };
};

const open = async (shown: string): Promise<void> => {
  fireEvent.click(await findDialogTriggerByText(shown));
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

const showDetails = async (): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Details" }));
  await screen.findAllByRole("term");
};

const openDownload = async (): Promise<void> => {
  await showDetails();
  fireEvent.click(getIconButton(document.body, "download"));
};

const summaryOf = (c: certificate.Certificate): string => {
  const summary = Secret.Validity.summarizeCertificate(c, TimeStamp.now());
  if (summary == null) throw new Error(`${c.name} has no details`);
  return summary.text;
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
    const option = await findOption("certificate", c.name);
    within(option).getByText(
      Secret.Validity.describeCertificate(c, TimeStamp.now()).text,
    );
  });

  it("shows no prompt to pick a certificate in preview", async () => {
    await renderField("", client, "preview");
    await screen.findByText("None");
    expect(screen.queryByText("Select a certificate")).toBeNull();
  });

  it("shows that the certificate is not found for a key that names none", async () => {
    await renderField(uuid.create());
    await findDialogTriggerByText("Certificate not found");
  });

  it("shows that the certificate is unavailable to a user who cannot retrieve it", async () => {
    const c = await createStored();
    await renderField(c.key, await createTestClientWithGrants(client));
    await findDialogTriggerByText("Unavailable certificate");
    expect(screen.queryByText(c.name)).toBeNull();
  });

  it("sets the value to the key of the picked certificate and shows its name and expiry", async () => {
    const c = await createStored();
    const { onChange } = await renderField();
    await open("Select a certificate");
    fireEvent.click(await findOption("certificate", c.name));
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith(c.key));
    const trigger = await findDialogTriggerByText(c.name);
    expect(trigger.textContent).toBe(`${c.name}${summaryOf(c)}`);
  });

  it("shows the expiry of a certificate in the trigger and not under the field", async () => {
    const c = await createStored();
    await renderField(c.key);
    const trigger = await findDialogTriggerByText(c.name);
    const badge = within(trigger).getByText(summaryOf(c));
    expect(screen.getAllByText(summaryOf(c))).toEqual([badge]);
  });

  it("shows the details of the certificate only while they are open", async () => {
    const c = await createStored();
    await renderField(c.key);
    await screen.findByRole("button", { name: "Details" });
    expect(screen.queryByRole("term")).toBeNull();
    await showDetails();
    expect(readDetails()).toEqual({
      Subject: c.details.subject,
      Issuer: c.details.issuer,
      Validity: Secret.Validity.describeCertificate(c, TimeStamp.now()).text,
      "SHA-256": fingerprintOf(c),
    });
    fireEvent.click(screen.getByRole("button", { name: "Hide details" }));
    expect(screen.queryByRole("term")).toBeNull();
    screen.getByRole("button", { name: "Details" });
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

  it("shows the path of a file certificate under the field with no badge and no download", async () => {
    const r = await client.racks.create({ name: uniqueName("rack") });
    const c = await client.certificates.set({
      name: uniqueName("file"),
      source: "file",
      certificatePath: "/etc/synnax/client.crt",
      rack: r.key,
    });
    created.push(c.key);
    await renderField(c.key);
    await screen.findByText("/etc/synnax/client.crt");
    expect((await findDialogTriggerByText(c.name)).textContent).toBe(c.name);
    await showDetails();
    expect(readDetails()).toEqual({ Path: "/etc/synnax/client.crt" });
    expect(queryIconButton(document.body, "download")).toBeNull();
  });

  it("opens the create dialog named after the search", async () => {
    await renderField();
    await open("Select a certificate");
    const name = uniqueName("searched");
    fireEvent.change(await screen.findByPlaceholderText("Search certificates..."), {
      target: { value: name },
    });
    fireEvent.click(
      await screen.findByRole("option", { name: `Create certificate "${name}"` }),
    );
    expect(await screen.findByLabelText("Certificate name")).toHaveProperty(
      "value",
      name,
    );
  });

  it("hides the create option for a certificate name in another case", async () => {
    const c = await createStored();
    await renderField();
    await open("Select a certificate");
    fireEvent.change(await screen.findByPlaceholderText("Search certificates..."), {
      target: { value: `${c.name}x` },
    });
    await screen.findByRole("option", { name: `Create certificate "${c.name}x"` });
    await findOption("certificate", c.name, c.name.toUpperCase());
    expect(screen.queryByRole("option", { name: /^Create certificate/ })).toBeNull();
  });

  it("offers no new certificate to a user who cannot create certificates", async () => {
    const c = await createStored();
    const reader = await createTestClientWithGrants(client, {
      retrieve: [certificate.TYPE_RESOURCE_ID],
    });
    await awaitGranted(reader, certificate.TYPE_RESOURCE_ID, "retrieve");
    await renderField("", reader);
    await open("Select a certificate");
    await findOption("certificate", c.name);
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

  it("requires a certificate before it saves", async () => {
    const { onChange } = await renderField();
    await openCreate();
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
});
