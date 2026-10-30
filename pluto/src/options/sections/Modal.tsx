import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactNode } from "react";

import {
  CHANNELS,
  cls,
  CrumbTitle,
  ESCAPE,
  ModalTitle,
  noop,
  Option,
  SAVE,
  SaveHint,
  Scrim,
  Section,
  StaticModal,
} from "@/options/common";
import { DataTypeSelect, Field } from "@/options/samples";

export type HeaderStyle = "bar" | "bar-title" | "crumb-title";
export type FooterStyle = "today" | "divider" | "plain" | "tint";

const IndexSelect = () => (
  <Select.Simple<string>
    resourceName="channel"
    value={undefined as unknown as string}
    onChange={noop}
    icon={<Icon.Channel />}
    allowNone
    full="x"
  >
    {CHANNELS.slice(-1).map((c) => (
      <Select.Item key={c} itemKey={c}>
        {c}
      </Select.Item>
    ))}
  </Select.Simple>
);

export const SwitchToggles = () => (
  <>
    <Field label="Virtual">
      <Input.Switch value={false} onChange={noop} />
    </Field>
    <Field label="Is index">
      <Input.Switch value={false} onChange={noop} />
    </Field>
  </>
);

const CreateMore = () => (
  <Flex.Box x align="center" gap="small">
    <Input.Switch value={false} onChange={noop} />
    <Text.Text color={9}>Create more</Text.Text>
  </Flex.Box>
);

interface ChannelCreateModalProps {
  header?: HeaderStyle;
  footer?: FooterStyle;
  toggles?: ReactNode;
  togglesBelow?: ReactNode;
  className?: string;
}

export const ChannelCreateModal = ({
  header = "bar",
  footer = "today",
  toggles = <SwitchToggles />,
  togglesBelow,
  className,
}: ChannelCreateModalProps) => (
  <StaticModal
    className={cls(
      "opt-create",
      header !== "bar" || footer !== "today" ? "opt-modal-inset" : undefined,
      className,
    )}
  >
    {header === "crumb-title" ? (
      <CrumbTitle icon={<Icon.Channel />} crumbs={["Channel", "Create"]} title="Create channel" />
    ) : (
      <Modal.Header icon={<Icon.Channel />}>{["Channel", "Create"]}</Modal.Header>
    )}
    <Modal.Body>
      <Flex.Box y gap="large">
        {header === "bar-title" && <ModalTitle>Create channel</ModalTitle>}
        <Field label="Name" required>
          <Input.Text level="h2" variant="text" placeholder="Name" value="" onChange={noop} />
        </Field>
        <Flex.Box x gap="large" align="start">
          {toggles}
          <Field label="Data type" required grow>
            <DataTypeSelect />
          </Field>
        </Flex.Box>
        {togglesBelow}
        <Field label="Index">
          <IndexSelect />
        </Field>
      </Flex.Box>
    </Modal.Body>
    {footer === "today" ? (
      <Modal.Footer>
        <SaveHint />
        <Nav.Bar.End align="center" gap="large">
          <CreateMore />
          <Button.Button variant="filled">Create</Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    ) : (
      <Modal.Footer className={cls(`opt-footer-${footer}`)}>
        <Nav.Bar.Start align="center">
          <CreateMore />
        </Nav.Bar.Start>
        <Nav.Bar.End align="center" gap="medium">
          <Button.Button variant="outlined" triggerIndicator={ESCAPE}>
            Cancel
          </Button.Button>
          <Button.Button variant="filled" triggerIndicator={SAVE}>
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    )}
  </StaticModal>
);

export const ModalSection = () => (
  <Section
    id="modal"
    n={6}
    title="Modal header and footer"
    summary="Sets how a modal names its subject and how its footer holds actions. Alternatives share one 24 px inset and put the shortcut on the primary button."
  >
    <Option
      letter="A"
      title="Today"
      desc="13 px breadcrumb bar. Divider footer with a separate shortcut hint at the left."
    >
      <Scrim>
        <ChannelCreateModal />
      </Scrim>
    </Option>
    <Option
      letter="B"
      title="Bar plus title, divider footer"
      desc="Breadcrumb bar kept; a 16 px title opens the body. Divider footer; Cancel and Create carry their keys."
    >
      <Scrim>
        <ChannelCreateModal header="bar-title" footer="divider" />
      </Scrim>
    </Option>
    <Option
      letter="C"
      title="Crumb over title, plain footer"
      desc="Small breadcrumb above a 16 px title, no bar. Plain footer with no divider."
    >
      <Scrim>
        <ChannelCreateModal header="crumb-title" footer="plain" />
      </Scrim>
    </Option>
    <Option
      letter="D"
      title="Crumb over title, tinted footer"
      desc="Small breadcrumb above a 16 px title, no bar. Footer is a tinted strip with a hairline."
    >
      <Scrim>
        <ChannelCreateModal header="crumb-title" footer="tint" />
      </Scrim>
    </Option>
  </Section>
);
