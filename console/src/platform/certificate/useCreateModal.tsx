// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, type rack, type secret, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Certificate, Synnax } from "@synnaxlabs/pluto";
import { deep, primitive, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import { readCertificate, readPrivateKey } from "@/platform/certificate/pem";
import { PEMInput } from "@/platform/certificate/PEMInput";
import { FileBrowser } from "@/platform/file-browser";
import { Modals } from "@/platform/modals";
import { Validity } from "@/platform/secret/validity";
import { Triggers } from "@/platform/triggers";

// The form holds the fields of both sources, so a switch keeps what was entered.
const fieldsZ = z.object({
  name: z.string().trim().min(1, "Name is required"),
  pem: z.string(),
  privateKeyPem: z.string(),
  certificatePath: z.string(),
  privateKeyFile: z.string(),
});

const formZ = z.discriminatedUnion("source", [
  fieldsZ.extend({
    source: z.literal("stored"),
    pem: z.string().trim().min(1, "Certificate is required"),
  }),
  fieldsZ.extend({
    source: z.literal("file"),
    certificatePath: z.string().trim().min(1, "Certificate file is required"),
  }),
]);

type FormValues = z.infer<typeof formZ>;

const path = deep.path<FormValues>();

const CERTIFICATE_EXTENSIONS = ["pem", "crt", "cer", "der"];

const KEY_EXTENSIONS = ["pem", "key"];

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  level: "h2",
  variant: "text",
  placeholder: "Certificate name",
  "aria-label": "Certificate name",
};

const FOCUSED_NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  ...NAME_INPUT_PROPS,
  autoFocus: true,
};

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

const StoredFields = ({ named }: { named: boolean }): ReactElement => (
  <>
    <Form.Field<string> path={path.pem} label="Certificate" emptiable>
      {({ value, onChange }) => (
        <PEMInput
          value={value}
          onChange={onChange}
          title="Select certificate"
          extensions={CERTIFICATE_EXTENSIONS}
          read={readCertificate}
          placeholder="-----BEGIN CERTIFICATE-----"
          autoFocus={named}
        />
      )}
    </Form.Field>
    <Form.Field<string> path={path.privateKeyPem} label="Private key" emptiable>
      {({ value, onChange }) => (
        <PEMInput
          value={value}
          onChange={onChange}
          title="Select private key"
          extensions={KEY_EXTENSIONS}
          read={readPrivateKey}
          placeholder="-----BEGIN PRIVATE KEY-----"
        />
      )}
    </Form.Field>
  </>
);

interface FileFieldsProps {
  rack: rack.Key;
  picked: FileBrowser.Entry | null;
  onPick: (entry: FileBrowser.Entry) => void;
}

const FileFields = ({ rack, picked, onPick }: FileFieldsProps): ReactElement => {
  const browse = FileBrowser.useModal();
  const now = useNow(nextMinute);
  return (
    <>
      <Form.Field<string>
        path={path.certificatePath}
        label="Certificate file"
        emptiable
      >
        {({ value, onChange }) => {
          // A path typed by hand gets its details at the first connect.
          const details = picked?.path === value ? picked.certificate : undefined;
          const validity =
            details == null ? null : Validity.describeDetails(details, now);
          const handleBrowse = async () => {
            const entry = await browse({ rack, path: value });
            if (entry == null) return;
            onPick(entry);
            onChange(entry.path);
          };
          return (
            <Flex.Box y gap="tiny" grow>
              <Flex.Box x gap="small" align="center">
                <Input.Text
                  value={value}
                  onChange={onChange}
                  placeholder="/etc/ssl/certs/client.crt"
                  spellCheck={false}
                  autoComplete="off"
                  grow
                />
                <Button.Button
                  variant="outlined"
                  tooltip="Browse"
                  onClick={() => void handleBrowse()}
                >
                  <Icon.Folder />
                </Button.Button>
              </Flex.Box>
              {details != null && validity != null && (
                <Text.Text level="small" color={9} status={validity.status}>
                  {`${details.subject} · ${validity.text}`}
                </Text.Text>
              )}
            </Flex.Box>
          );
        }}
      </Form.Field>
      <Form.Field<string> path={path.privateKeyFile} label="Private key file" emptiable>
        {({ value, onChange }) => (
          <FileBrowser.SelectSecretFile
            rack={rack}
            value={value}
            onChange={onChange}
            allowNone
          />
        )}
      </Form.Field>
    </>
  );
};

export interface CreateModalParams {
  /** The name a created certificate starts with. */
  name?: string;
  /** The rack whose files the modal browses. Zero offers no file. */
  rack?: rack.Key;
}

const INITIAL_VALUES: Omit<FormValues, "name"> = {
  source: "stored",
  pem: "",
  privateKeyPem: "",
  certificatePath: "",
  privateKeyFile: "",
};

const Create = ({
  close,
  name: initialName = "",
  rack = 0,
}: Modals.ContentProps<CreateModalParams, certificate.Key>): ReactElement => {
  const client = Synnax.use();
  const form = Form.use({
    values: { name: initialName, ...INITIAL_VALUES },
    schema: formZ,
  });
  const source = Form.useFieldValue<secret.Source>(path.source, { ctx: form });
  const [picked, setPicked] = useState<FileBrowser.Entry | null>(null);
  const { update, variant } = Certificate.useCreate({
    afterSuccess: ({ data }) => close(data.key),
  });
  const busy = variant === "loading";
  Modals.useBusy(busy);
  // A given name moves focus to the certificate.
  const named = initialName.length > 0;
  const handleSave = () => {
    if (!form.validate()) return;
    const values = form.value();
    const name = values.name.trim();
    if (values.source === "stored") {
      const { pem, privateKeyPem } = values;
      update({
        name,
        source: "stored",
        pem,
        privateKeyPem: privateKeyPem.trim() === "" ? undefined : privateKeyPem,
      });
      return;
    }
    const certificatePath = values.certificatePath.trim();
    update({
      name,
      source: "file",
      certificatePath,
      privateKeyFile: values.privateKeyFile === "" ? undefined : values.privateKeyFile,
      details: picked?.path === certificatePath ? picked.certificate : undefined,
      rack,
    });
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Certificate />}>Certificate.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof formZ> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            padHelpText={false}
            inputProps={
              initialName === "" ? FOCUSED_NAME_INPUT_PROPS : NAME_INPUT_PROPS
            }
          />
          {primitive.isNonZero(rack) && (
            <Form.Field<secret.Source> path={path.source} label="Source">
              {({ value, onChange }) => (
                <Select.Buttons<secret.Source> value={value} onChange={onChange}>
                  <Select.Item itemKey="stored">
                    <Icon.Import />
                    Paste or drop
                  </Select.Item>
                  <Select.Item itemKey="file">
                    <Icon.Rack />
                    File on Driver
                  </Select.Item>
                </Select.Buttons>
              )}
            </Form.Field>
          )}
          {source === "file" ? (
            <FileFields rack={rack} picked={picked} onPick={setPicked} />
          ) : (
            <StoredFields named={named} />
          )}
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            onClick={() => close()}
            disabled={busy}
            triggerIndicator={PTriggers.ESCAPE}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            onClick={handleSave}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a modal that creates a certificate, with an optional private key. It stores
 * pasted PEM or a dropped file, or names a file on the rack. Resolves with the key of
 * the certificate, or null on cancel.
 */
export const useCreateModal = Modals.createPrompt<certificate.Key, CreateModalParams>(
  Create,
);
