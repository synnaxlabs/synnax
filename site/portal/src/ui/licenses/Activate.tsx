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
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Steps } from "@synnaxlabs/lyra/steps";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, type ReactNode, useState } from "react";
import { z } from "zod";

import { read } from "@/server/license/fingerprint";
import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { type Held, type Owned } from "@/server/license/list";
import { post, save } from "@/ui/api";
import { Auth } from "@/ui/auth";
import { useLogOut } from "@/ui/clerk";
import { CONTACT_URL } from "@/ui/Enterprise";
import { Tile } from "@/ui/Tile";
import { useAction } from "@/ui/useAction";

/** Machine is the machine an activation link came with. */
export interface Machine {
  fingerprint: string[];
  /** name is the machine's name on its seat, or the name its link suggested. */
  name: string;
  /** seat is the license this machine holds a seat on, if any. */
  seat: string | null;
}

export interface ActivateProps {
  choices: (Owned & Held)[];
  /** selected is the license the page opens on. */
  selected: string | null;
  /** machine is the machine the link came with, or null when it came with none. */
  machine: Machine | null;
  /** damaged is true when the link held a fingerprint that the page could not read. */
  damaged: boolean;
  /** handoff is the code of the Console waiting for the license key, if any. */
  handoff: string | null;
  /** email is the logged-in user's email. */
  email: string;
}

interface Activated {
  key: string;
  activation: string;
  filename: string;
}

interface Done extends Activated {
  name: string;
  license: string;
}

const schema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name the machine")
    .max(MAX_NAME_LENGTH, "Use a shorter name"),
  fingerprint: z.string().superRefine((text, ctx) => {
    try {
      read(text);
    } catch (err) {
      ctx.addIssue({ code: "custom", message: (err as Error).message });
    }
  }),
});

const DAMAGED =
  "The link holds no fingerprint the portal can read. Open it again from the Console, " +
  "or paste the fingerprint below.";

const adapters = (count: number): string =>
  `${count} network ${count === 1 ? "adapter" : "adapters"}`;

const backToPortal = (
  <Button.Button variant="text" size="small" textColor={9} href="/">
    Back to the portal
  </Button.Button>
);

/**
 * Activate is the page that gives a machine its license key. The Console and the
 * Core's log open it with the machine's fingerprint, and the Console collects the key
 * on its own when the link carries a handoff code.
 */
export const Activate = ({
  choices,
  selected,
  machine,
  damaged,
  handoff,
  email,
}: ActivateProps): ReactElement => {
  const [entering, setEntering] = useState(damaged);
  const [done, setDone] = useState<Done | null>(null);
  if (choices.length === 0)
    return (
      <NoLicense email={email} machine={machine != null} handoff={handoff != null} />
    );
  if (done != null) return <Finished done={done} handoff={handoff != null} />;
  if (machine == null && !entering) return <Start onEnter={() => setEntering(true)} />;
  return (
    <Fields
      choices={choices}
      selected={selected ?? choices[0].license.key}
      machine={machine}
      damaged={damaged}
      handoff={handoff}
      onDone={setDone}
    />
  );
};

interface NoLicenseProps {
  email: string;
  machine: boolean;
  handoff: boolean;
}

/** NoLicense shows a user with no license to activate where to go next. */
const NoLicense = ({ email, machine, handoff }: NoLicenseProps): ReactElement => {
  const logOut = useLogOut();
  let forward = "Ask them to invite you to their organization.";
  if (handoff)
    forward =
      "Send them this link. When they activate the machine, the Console gets the " +
      "license key on its own.";
  else if (machine)
    forward = "Send them this link. It opens with the machine filled in.";
  return (
    <Auth.Card
      icon={
        <Tile>
          <Icon.License />
        </Tile>
      }
      title="No license to activate"
      description={`Logged in as ${email}`}
      footer={backToPortal}
    >
      <Flex.Box y gap="large">
        <Way title="Someone else manages your licenses?" description={forward}>
          {machine && (
            <Button.Copy
              variant="outlined"
              full="x"
              justify="center"
              text={() => window.location.href}
              successMessage="Copied link"
            >
              Copy link
            </Button.Copy>
          )}
        </Way>
        <Way
          title="Logged in with the wrong account?"
          description="Log in with the account your organization uses."
        >
          <Button.Button
            variant="outlined"
            full="x"
            justify="center"
            onClick={logOut}
            disabled={logOut == null}
          >
            Log in with another account
          </Button.Button>
        </Way>
        <Way
          title="Need a license?"
          description="Synnax Labs sets up organizations and issues licenses."
        >
          <Button.Button
            variant="outlined"
            full="x"
            justify="center"
            href={CONTACT_URL}
          >
            <Icon.Feedback />
            Talk to us
          </Button.Button>
        </Way>
      </Flex.Box>
    </Auth.Card>
  );
};

const Start = ({ onEnter }: { onEnter: () => void }): ReactElement => (
  <Auth.Card
    icon={
      <Tile>
        <Icon.Computer />
      </Tile>
    }
    title="Activate a machine"
    description="Start on the machine that runs the Core"
    footer={backToPortal}
  >
    <Steps.List level="small">
      <Steps.Item>
        Open the Console and log in to the Core. It shows{" "}
        <b>This Core needs a license</b>.
      </Steps.Item>
      <Steps.Item>
        Click <b>Activate on the portal</b>. This page opens with the machine filled in.
      </Steps.Item>
    </Steps.List>
    <Text.Text level="small" color={9}>
      A Core that no one logs in to prints the same link in its log when it starts.
    </Text.Text>
    <Button.Button variant="outlined" full="x" justify="center" onClick={onEnter}>
      Enter a fingerprint instead
    </Button.Button>
  </Auth.Card>
);

interface FieldsProps {
  choices: (Owned & Held)[];
  selected: string;
  machine: Machine | null;
  damaged: boolean;
  handoff: string | null;
  onDone: (done: Done) => void;
}

const Fields = ({
  choices,
  selected,
  machine,
  damaged,
  handoff,
  onDone,
}: FieldsProps): ReactElement => {
  const [licenseKey, setLicenseKey] = useState(selected);
  const [editing, setEditing] = useState(machine == null);
  const methods = Form.use({
    values: {
      name: machine?.name ?? "",
      fingerprint: machine?.fingerprint.join("\n") ?? "",
    },
    schema,
  });
  const choice = choices.find((c) => c.license.key === licenseKey) ?? choices[0];
  const held = !editing && machine?.seat === choice.license.key;
  const full = !held && choice.seats >= choice.license.nodes;
  const action = useAction(async () => {
    if (!methods.validate()) return;
    const { name, fingerprint } = methods.value();
    const res = await post<Activated>(`/api/licenses/${choice.license.key}/activate`, {
      name,
      fingerprint: read(fingerprint),
      handoff: handoff ?? undefined,
    });
    onDone({ ...res, name: name.trim(), license: choice.license.key });
  });
  let seats: ReactNode = (
    <Text.Text level="small" color={9}>
      {choice.seats} of {choice.license.nodes} seats in use
    </Text.Text>
  );
  if (held)
    seats = (
      <Text.Text level="small" color={9}>
        This machine already holds a seat on this license. Activating it again keeps the
        seat and issues a new key.
      </Text.Text>
    );
  else if (full)
    seats = (
      <Flex.Box x align="center" justify="between" gap="small">
        <Text.Text level="small" status="warning">
          Every seat is taken. Only a machine that holds one can activate.
        </Text.Text>
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          href={`/licenses/${choice.license.key}`}
        >
          Release a machine
          <Icon.Arrow.Right />
        </Button.Button>
      </Flex.Box>
    );
  return (
    <Auth.Card
      icon={
        <Tile>
          <Icon.Computer />
        </Tile>
      }
      title={machine == null ? "Activate a machine" : "Activate this machine"}
      description={
        handoff == null
          ? "Name the machine, then get its license key"
          : "The Console gets the license key when you finish"
      }
      error={action.error ?? (damaged && editing ? DAMAGED : null)}
      footer={backToPortal}
    >
      <Form.Form<typeof schema> {...methods}>
        <Flex.Box y gap="large">
          {editing ? (
            <Fingerprint />
          ) : (
            <Received
              count={machine?.fingerprint.length ?? 0}
              onChange={() => setEditing(true)}
            />
          )}
          <Flex.Box y gap="small">
            <Text.Text level="small" color={9}>
              License
            </Text.Text>
            {choices.length === 1 ? (
              <Text.Text level="p" color={11}>
                {choice.license.label}
              </Text.Text>
            ) : (
              <Select.Simple<string>
                resourceName="License"
                value={choice.license.key}
                onChange={setLicenseKey}
              >
                {choices.map(({ license, organization }) => (
                  <Select.Item key={license.key} itemKey={license.key}>
                    {`${license.label} (${organization.name})`}
                  </Select.Item>
                ))}
              </Select.Simple>
            )}
            {seats}
          </Flex.Box>
          <Form.TextField
            path="name"
            label="Machine name"
            inputProps={{
              autoFocus: machine != null && machine.name === "",
              placeholder: "Test stand, site B",
            }}
          />
        </Flex.Box>
      </Form.Form>
      <Button.Button
        variant="filled"
        size="large"
        full="x"
        justify="center"
        status={action.loading ? "loading" : undefined}
        onClick={action.run}
        trigger={editing ? undefined : ["Enter"]}
      >
        Activate
        <Icon.Arrow.Right />
      </Button.Button>
    </Auth.Card>
  );
};

interface ReceivedProps {
  count: number;
  onChange: () => void;
}

/** Received shows the fingerprint that came with the link. */
const Received = ({ count, onChange }: ReceivedProps): ReactElement => (
  <Flex.Box y gap="small">
    <Text.Text level="small" color={9}>
      Machine
    </Text.Text>
    <Flex.Box
      x
      align="center"
      justify="between"
      gap="medium"
      className="portal-received"
    >
      <Text.Text level="p" color={11} gap="small">
        <Icon.Check className="portal-received__check" />
        Fingerprint received, {adapters(count)}
      </Text.Text>
      <Button.Button variant="text" size="small" textColor={9} onClick={onChange}>
        Change
      </Button.Button>
    </Flex.Box>
  </Flex.Box>
);

/** Fingerprint takes a fingerprint, or the activation link that carries one. */
const Fingerprint = (): ReactElement => {
  const text = Form.useFieldValue<string>("fingerprint") ?? "";
  let found: string | null = null;
  try {
    found = adapters(read(text).length);
  } catch {
    // The field reports the problem when the form is submitted.
  }
  return (
    <Flex.Box y gap="tiny">
      <Form.Field<string> path="fingerprint" label="Fingerprint">
        {(p) => (
          <Input.Text
            {...p}
            area
            autoFocus
            spellCheck={false}
            placeholder="Paste the activation link or the fingerprint from the Core's log"
            className="portal-fingerprint"
          />
        )}
      </Form.Field>
      {found != null && (
        <Text.Text level="small" color={9} gap="small">
          <Icon.Check className="portal-received__check" />
          {found}
        </Text.Text>
      )}
    </Flex.Box>
  );
};

interface FinishedProps {
  done: Done;
  /** handoff is true when a Console waits to collect the key. */
  handoff: boolean;
}

const Finished = ({ done, handoff }: FinishedProps): ReactElement => {
  const download = (): void =>
    save(new Blob([done.key], { type: "text/plain" }), done.filename);
  const keyButtons = (
    <Flex.Box x gap="small" full="x">
      <Button.Copy
        variant="outlined"
        full="x"
        justify="center"
        text={done.key}
        successMessage="Copied license key"
      >
        Copy key
      </Button.Copy>
      <Button.Button variant="outlined" full="x" justify="center" onClick={download}>
        <Icon.Download />
        Download key file
      </Button.Button>
    </Flex.Box>
  );
  return (
    <Auth.Card
      icon={
        <Tile status="success">
          <Icon.Check />
        </Tile>
      }
      title={`${done.name} is activated`}
      description={
        handoff
          ? "Go back to the Console. It gets the license key and activates the Core on its own."
          : "Give this license key to the Core"
      }
      footer={
        <Flex.Box x justify="center" gap="small" wrap>
          <Button.Button
            variant="text"
            size="small"
            textColor={9}
            href={`/licenses/${done.license}`}
          >
            View license
          </Button.Button>
          <Button.Button
            variant="text"
            size="small"
            textColor={9}
            href="/licenses/activate"
          >
            Activate another machine
          </Button.Button>
        </Flex.Box>
      }
    >
      {handoff ? (
        <Flex.Box y gap="medium">
          <Text.Text level="small" color={9}>
            If the Console cannot reach the portal, paste the key on its activation
            screen.
          </Text.Text>
          {keyButtons}
        </Flex.Box>
      ) : (
        <Flex.Box y gap="huge">
          {keyButtons}
          <Flex.Box y gap="large">
            <Way
              title="In the Console"
              description="Paste the key on the activation screen, or load the key file."
            />
            <Way title="At start">
              <Command text={`synnax start --license-file=${done.filename}`} />
            </Way>
            <Way
              title="In an environment variable"
              description="For Docker, Kubernetes, and CI."
            >
              <Command text={`export SYNNAX_LICENSE_KEY="$(cat ${done.filename})"`} />
            </Way>
          </Flex.Box>
        </Flex.Box>
      )}
    </Auth.Card>
  );
};

interface WayProps {
  title: string;
  description?: string;
  children?: ReactNode;
}

/** Way is one way forward: a title, an optional description, and an action. */
const Way = ({ title, description, children }: WayProps): ReactElement => (
  <Flex.Box y gap="small">
    <Flex.Box y gap="tiny">
      <Text.Text level="small" weight={500} color={11}>
        {title}
      </Text.Text>
      {description != null && (
        <Text.Text level="small" color={9}>
          {description}
        </Text.Text>
      )}
    </Flex.Box>
    {children}
  </Flex.Box>
);

/** Command shows a shell command with a button that copies it. */
const Command = ({ text }: { text: string }): ReactElement => (
  <Flex.Box x align="center" justify="between" gap="small" className="portal-command">
    <Text.Text level="small" variant="code" color={10} overflow="fade">
      {text}
    </Text.Text>
    <Button.Copy variant="text" size="small" text={text} tooltip="Copy" />
  </Flex.Box>
);
