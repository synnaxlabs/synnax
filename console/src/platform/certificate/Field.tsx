// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, type rack } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Access, Certificate } from "@synnaxlabs/pluto";
import { primitive, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Details } from "@/platform/certificate/Details";
import { useCreateModal } from "@/platform/certificate/useCreateModal";
import { Picker } from "@/platform/picker";
import { Secret } from "@/platform/secret";

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

interface ControlProps extends Input.Control<certificate.Key> {
  defaultName: string;
  rack: rack.Key;
  preview?: boolean;
}

const Control = ({ value, defaultName, rack, ...rest }: ControlProps): ReactElement => {
  const list = Certificate.useList({ query: { where: Secret.whereOnRack(rack) } });
  const result = Certificate.useResult(primitive.isZero(value) ? null : { key: value });
  const canCreate = Access.useCreateGranted(certificate.TYPE_RESOURCE_ID);
  const openCreate = useCreateModal();
  const now = useNow(nextMinute);
  const describe = (c: certificate.Certificate): Picker.Description =>
    Secret.Validity.describeCertificate(c, now);
  const renderValidity = (c: certificate.Certificate): ReactElement | null => {
    const validity = Secret.Validity.summarizeCertificate(c, now);
    if (validity == null) return null;
    return <Badge.Badge status={validity.status}>{validity.text}</Badge.Badge>;
  };
  return (
    <Flex.Box y gap="small" grow>
      <Picker.Select<certificate.Certificate>
        value={value}
        resourceName="certificate"
        icon={Icon.Certificate}
        describe={describe}
        renderEndContent={renderValidity}
        list={list}
        result={result}
        onCreate={
          canCreate
            ? (name) => openCreate({ name: name === "" ? defaultName : name, rack })
            : undefined
        }
        {...rest}
      />
      {result.variant === "success" && (
        <Details certificate={result.data} now={now} rack={rack} />
      )}
    </Flex.Box>
  );
};

export interface FieldProps extends Omit<Form.FieldProps<certificate.Key>, "children"> {
  /** The name a created certificate starts with, such as "Plant CA certificate". */
  defaultName: string;
  /** The rack of the device. The field lists only the file items of this rack. */
  rack: rack.Key;
}

/**
 * A form field that holds the key of a certificate and shows its details. It selects a
 * certificate, or creates one that starts with the default name and selects it.
 */
export const Field = ({ defaultName, rack, ...rest }: FieldProps): ReactElement => (
  <Form.Field<certificate.Key> {...rest}>
    {({ value, onChange, preview }) => (
      <Control
        value={value}
        onChange={onChange}
        defaultName={defaultName}
        rack={rack}
        preview={preview}
      />
    )}
  </Form.Field>
);
