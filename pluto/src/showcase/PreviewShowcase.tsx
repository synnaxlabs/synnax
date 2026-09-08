// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";
import { z } from "zod";

import { Button } from "@/button";
import { Component } from "@/component";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { Select } from "@/select";
import { Text } from "@/text";

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

const TYPE_KEYS = ["analog", "digital"];

const SelectType = ({
  preview,
  value: initial,
}: {
  preview: boolean;
  value: string | undefined;
}) => {
  const [value, setValue] = useState<string | undefined>(initial);
  return (
    <Select.Buttons
      keys={TYPE_KEYS}
      value={value}
      onChange={setValue}
      preview={preview}
    >
      <Select.Button itemKey="analog">Analog</Select.Button>
      <Select.Button itemKey="digital">Digital</Select.Button>
    </Select.Buttons>
  );
};

const SelectAlign = ({ preview }: { preview: boolean }) => {
  const [value, setValue] = useState<string>("x-center");
  return (
    <Select.Buttons
      keys={["x-center", "y-center", "x-left"]}
      value={value}
      onChange={setValue}
      preview={preview}
    >
      <Select.Button itemKey="x-center">
        <Icon.Align.XCenter />
      </Select.Button>
      <Select.Button itemKey="y-center">
        <Icon.Align.YCenter />
      </Select.Button>
      <Select.Button itemKey="x-left">
        <Icon.Align.Left />
      </Select.Button>
    </Select.Buttons>
  );
};

const PORTS = [
  { key: "ai0", name: "AI0" },
  { key: "ai1", name: "AI1" },
  { key: "ai2", name: "AI2" },
];

const SelectSingleStatic = ({
  preview,
  value: initial,
}: {
  preview: boolean;
  value: string | undefined;
}) => {
  const [value, setValue] = useState<string | undefined>(initial);
  return (
    <Select.Static
      resourceName="port"
      data={PORTS}
      value={value ?? ""}
      onChange={setValue}
      allowNone
      preview={preview}
    />
  );
};

const LABELS = [
  { key: "hotfire", name: "Hotfire" },
  { key: "coldflow", name: "Coldflow" },
  { key: "abort", name: "Abort" },
];

const labelListItem = Component.renderProp((props: List.ItemProps<string>) => {
  const item = List.useItem<string, record.KeyedNamed>(props.itemKey);
  return <Select.ListItem {...props}>{item?.name}</Select.ListItem>;
});

const SelectMultipleStatic = ({
  preview,
  value: initial,
}: {
  preview: boolean;
  value: string[];
}) => {
  const [value, setValue] = useState<string[]>(initial);
  const { data, getItem } = List.useStaticData<string, record.KeyedNamed>({
    data: LABELS,
  });
  return (
    <Select.Multiple
      resourceName="label"
      data={data}
      getItem={getItem}
      value={value}
      onChange={setValue}
      preview={preview}
    >
      {labelListItem}
    </Select.Multiple>
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
            <Select.Buttons
              keys={TYPE_KEYS}
              value={value}
              onChange={onChange}
              preview={p}
            >
              <Select.Button itemKey="analog">Analog</Select.Button>
              <Select.Button itemKey="digital">Digital</Select.Button>
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
    <Row label="Date">
      {(p) => <Input.Date value={NOW} onChange={() => {}} preview={p} />}
    </Row>
    <Row label="Time">
      {(p) => <Input.Time value={NOW} onChange={() => {}} preview={p} />}
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
      {(p) => <SelectSingleStatic preview={p} value="ai1" />}
    </Row>
    <Row label="Select single none">
      {(p) => <SelectSingleStatic preview={p} value={undefined} />}
    </Row>
    <Row label="Select multiple tags">
      {(p) => <SelectMultipleStatic preview={p} value={["hotfire", "coldflow"]} />}
    </Row>
    <Row label="Select multiple empty">
      {(p) => <SelectMultipleStatic preview={p} value={[]} />}
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
