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
import { Text } from "@synnaxlabs/lyra/text";
import { TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";
import { z } from "zod";

interface RowProps {
  label: string;
  children: (preview: boolean) => ReactNode;
}

const Row = ({ label, children }: RowProps): ReactElement => (
  <Flex.Box
    x
    align="center"
    style={{ display: "grid", gridTemplateColumns: "22rem 1fr 1fr", gap: "2rem" }}
  >
    <Text.Text level="small" color={9}>
      {label}
    </Text.Text>
    <Flex.Box x align="center">
      {children(false)}
    </Flex.Box>
    <Flex.Box x align="center">
      {children(true)}
    </Flex.Box>
  </Flex.Box>
);

const SelectType = ({
  preview,
  value: initial,
}: {
  preview: boolean;
  value: string | undefined;
}) => {
  const [value, setValue] = useState<string | undefined>(initial);
  return (
    <Select.Buttons value={value} onChange={setValue} preview={preview}>
      <Select.Item itemKey="analog">Analog</Select.Item>
      <Select.Item itemKey="digital">Digital</Select.Item>
    </Select.Buttons>
  );
};

const SelectAlign = ({ preview }: { preview: boolean }) => {
  const [value, setValue] = useState<string>("x-center");
  return (
    <Select.Buttons value={value} onChange={setValue} preview={preview}>
      <Select.Item itemKey="x-center">
        <Icon.Align.XCenter />
      </Select.Item>
      <Select.Item itemKey="y-center">
        <Icon.Align.YCenter />
      </Select.Item>
      <Select.Item itemKey="x-left">
        <Icon.Align.Left />
      </Select.Item>
    </Select.Buttons>
  );
};

const SelectSimple = ({
  preview,
  value: initial,
}: {
  preview: boolean;
  value: string | undefined;
}) => {
  const [value, setValue] = useState<string | undefined>(initial);
  return (
    <Select.Simple<string>
      resourceName="port"
      value={value ?? ""}
      onChange={setValue}
      allowNone
      preview={preview}
    >
      <Select.Item itemKey="ai0">AI0</Select.Item>
      <Select.Item itemKey="ai1">AI1</Select.Item>
      <Select.Item itemKey="ai2">AI2</Select.Item>
    </Select.Simple>
  );
};

const LABELS = [
  { key: "hotfire", name: "Hotfire" },
  { key: "coldflow", name: "Coldflow" },
  { key: "abort", name: "Abort" },
];

const SelectMultiple = ({
  preview,
  value: initial,
}: {
  preview: boolean;
  value: string[];
}) => {
  const [value, setValue] = useState<string[]>(initial);
  return (
    <Select.Simple
      multiple
      resourceName="label"
      value={value}
      onChange={setValue}
      preview={preview}
    >
      {LABELS.map(({ key, name }) => (
        <Select.Item key={key} itemKey={key}>
          {name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

const NOW = Number(TimeStamp.now().valueOf());

const formSchema = z.object({
  name: z.string(),
  rate: z.number(),
  enabled: z.boolean(),
  type: z.string(),
});

const PreviewForm = ({ preview }: { preview: boolean }) => {
  const methods = Form.use({
    values: { name: "Pressure sensor", rate: 50, enabled: true, type: "digital" },
    schema: formSchema,
    mode: preview ? "preview" : "normal",
  });
  return (
    <Form.Form<typeof formSchema> {...methods}>
      <Flex.Box y gap="medium" grow>
        <Form.TextField path="name" label="Name" />
        <Form.NumericField path="rate" label="Sample rate" />
        <Form.SwitchField path="enabled" label="Data saving" />
        <Form.Field<string> path="type" label="Type">
          {({ value, onChange, preview: p }) => (
            <Select.Buttons value={value} onChange={onChange} preview={p}>
              <Select.Item itemKey="analog">Analog</Select.Item>
              <Select.Item itemKey="digital">Digital</Select.Item>
            </Select.Buttons>
          )}
        </Form.Field>
      </Flex.Box>
    </Form.Form>
  );
};

export const PreviewShowcase = (): ReactElement => (
  <Flex.Box y gap="large" style={{ padding: "3rem" }}>
    <Flex.Box
      x
      style={{ display: "grid", gridTemplateColumns: "22rem 1fr 1fr", gap: "2rem" }}
    >
      <Text.Text level="small" weight={500}>
        Component
      </Text.Text>
      <Text.Text level="small" weight={500}>
        Normal
      </Text.Text>
      <Text.Text level="small" weight={500}>
        Preview
      </Text.Text>
    </Flex.Box>
    <Row label="Button outlined">
      {(p) => <Button.Button preview={p}>Configure</Button.Button>}
    </Row>
    <Row label="Button filled">
      {(p) => (
        <Button.Button variant="filled" preview={p}>
          Save
        </Button.Button>
      )}
    </Row>
    <Row label="Toggle checked">
      {(p) => (
        <Button.Toggle value onChange={() => {}} preview={p}>
          Enabled
        </Button.Toggle>
      )}
    </Row>
    <Row label="Toggle unchecked">
      {(p) => (
        <Button.Toggle value={false} onChange={() => {}} preview={p}>
          Enabled
        </Button.Toggle>
      )}
    </Row>
    <Row label="Text with value">
      {(p) => <Input.Text value="gse_pressure_1" onChange={() => {}} preview={p} />}
    </Row>
    <Row label="Text empty + placeholder">
      {(p) => (
        <Input.Text value="" onChange={() => {}} placeholder="Name" preview={p} />
      )}
    </Row>
    <Row label="Text with end content">
      {(p) => (
        <Input.Text value="120" onChange={() => {}} endContent="psi" preview={p} />
      )}
    </Row>
    <Row label="Numeric">
      {(p) => <Input.Numeric value={50} onChange={() => {}} preview={p} />}
    </Row>
    <Row label="DateTime">
      {(p) => <Input.DateTime value={NOW} onChange={() => {}} preview={p} />}
    </Row>
    <Row label="Switch on / off">
      {(p) => (
        <>
          <Input.Switch value onChange={() => {}} preview={p} />
          <Input.Switch value={false} onChange={() => {}} preview={p} />
        </>
      )}
    </Row>
    <Row label="Checkbox on / off">
      {(p) => (
        <>
          <Input.Checkbox value onChange={() => {}} preview={p} />
          <Input.Checkbox value={false} onChange={() => {}} preview={p} />
        </>
      )}
    </Row>
    <Row label="Select buttons selected">
      {(p) => <SelectType preview={p} value="digital" />}
    </Row>
    <Row label="Select buttons none">
      {(p) => <SelectType preview={p} value={undefined} />}
    </Row>
    <Row label="Select buttons icons">{(p) => <SelectAlign preview={p} />}</Row>
    <Row label="Select single selected">
      {(p) => <SelectSimple preview={p} value="ai1" />}
    </Row>
    <Row label="Select single none">
      {(p) => <SelectSimple preview={p} value={undefined} />}
    </Row>
    <Row label="Select multiple tags">
      {(p) => <SelectMultiple preview={p} value={["hotfire", "coldflow"]} />}
    </Row>
    <Row label="Select multiple empty">
      {(p) => <SelectMultiple preview={p} value={[]} />}
    </Row>
    <Flex.Box x gap="huge" style={{ marginTop: "3rem" }}>
      <Flex.Box y grow>
        <Text.Text level="small" weight={500}>
          Form normal
        </Text.Text>
        <PreviewForm preview={false} />
      </Flex.Box>
      <Flex.Box y grow>
        <Text.Text level="small" weight={500}>
          Form preview
        </Text.Text>
        <PreviewForm preview />
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
