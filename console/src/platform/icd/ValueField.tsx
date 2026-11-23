// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { deep, TimeStamp } from "@synnaxlabs/x";
import type { FC, ReactElement } from "react";

type Kind = icd.JSONField["type"];

/** The raw value of a field of each kind. */
type Raw = { [K in Kind]: Extract<icd.WriteValue, { type: K }>["value"] };

interface RawFieldProps<V> {
  path: deep.Target<V | undefined>;
  label: string;
}

type KindFieldProps<K extends Kind> = RawFieldProps<Raw[K]>;

const NumberField = ({ path, label }: KindFieldProps<"float">): ReactElement => (
  <Form.NumericField path={path} label={label} padHelpText={false} emptiable={false} />
);

const TextField = ({ path, label }: RawFieldProps<string>): ReactElement => (
  <Form.TextField path={path} label={label} padHelpText={false} emptiable={false} />
);

const BOOLEANS = { false: false, true: true } as const;

type BooleanKey = keyof typeof BOOLEANS;

const BooleanField = ({ path, label }: KindFieldProps<"boolean">): ReactElement => (
  <Form.Field<boolean> path={deep.present(path)} label={label} padHelpText={false}>
    {({ value, onChange, preview }) => (
      <Select.Buttons<BooleanKey>
        value={String(value) as BooleanKey}
        onChange={(key: BooleanKey) => onChange(BOOLEANS[key])}
        preview={preview}
      >
        <Select.Item<BooleanKey> itemKey="false">False</Select.Item>
        <Select.Item<BooleanKey> itemKey="true">True</Select.Item>
      </Select.Buttons>
    )}
  </Form.Field>
);

const TimeStampField = ({ path, label }: KindFieldProps<"timestamp">): ReactElement => (
  <Form.Field<TimeStamp> path={deep.present(path)} label={label} padHelpText={false}>
    {({ value, onChange, preview }) => (
      <Input.DateTime
        value={value.nanoseconds}
        onChange={(ns: number) => onChange(new TimeStamp(ns))}
        preview={preview}
      />
    )}
  </Form.Field>
);

const FIELDS: { [K in Kind]: FC<KindFieldProps<K>> } = {
  float: NumberField,
  integer: TextField,
  unsigned: TextField,
  string: TextField,
  boolean: BooleanField,
  timestamp: TimeStampField,
  json: TextField,
  uuid: TextField,
  bytes: TextField,
};

export interface ValueFieldProps<K extends Kind> {
  /** The kind of the field that takes the value. */
  type: K;
  path: deep.Target<Raw[K] | undefined>;
  label?: string;
}

/** Edits a raw value of a field of the kind, such as its default or a fixed value. */
export const ValueField = <K extends Kind>({
  type,
  path,
  label = "Value",
}: ValueFieldProps<K>): ReactElement => {
  const Field: FC<KindFieldProps<K>> = FIELDS[type];
  return <Field path={path} label={label} />;
};
