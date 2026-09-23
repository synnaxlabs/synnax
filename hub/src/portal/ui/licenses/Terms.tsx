// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Input, Select, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { z } from "zod";

import { type License, type Term } from "@/server/db/schema";

const VERSION_PATTERN = /^\d+\.\d+$/;

/** termsSchema is the unrefined shape; apply checkTerms to get the term rules. */
export const termsSchema = z.object({
  label: z.string().trim().min(1, "Give the license a label"),
  term: z.enum(["subscription", "perpetual"]),
  nodes: z.number().int().min(1, "At least one machine"),
  channels: z.number().int().min(0, "0 for unlimited"),
  expiresAt: z.string(),
  maxVersion: z.string().trim(),
});

export interface TermsValues extends z.infer<typeof termsSchema> {}

/** checkTerms holds the rules that tie the term to the expiry and version ceiling. */
export const checkTerms = (ctx: z.core.ParsePayload<TermsValues>): void => {
  const v = ctx.value;
  if (v.term === "subscription" && v.expiresAt === "")
    ctx.issues.push({
      code: "custom",
      message: "A subscription needs an expiry date",
      path: ["expiresAt"],
      input: v.expiresAt,
    });
  if (v.term === "perpetual" && v.maxVersion === "")
    ctx.issues.push({
      code: "custom",
      message: "A perpetual license needs a maximum version",
      path: ["maxVersion"],
      input: v.maxVersion,
    });
  if (v.maxVersion !== "" && !VERSION_PATTERN.test(v.maxVersion))
    ctx.issues.push({
      code: "custom",
      message: "Use major.minor, like 0.62",
      path: ["maxVersion"],
      input: v.maxVersion,
    });
};

export const ZERO_TERMS: TermsValues = {
  label: "",
  term: "subscription",
  nodes: 1,
  channels: 0,
  expiresAt: "",
  maxVersion: "",
};

/** termsOf reads a license into the values the form edits. */
export const termsOf = (lic: License): TermsValues => ({
  label: lic.label,
  term: lic.term,
  nodes: lic.nodes,
  channels: lic.channels,
  expiresAt: lic.expiresAt?.toISOString().slice(0, 10) ?? "",
  maxVersion: lic.maxVersion ?? "",
});

const TERMS: { key: Term; name: string }[] = [
  { key: "subscription", name: "Subscription" },
  { key: "perpetual", name: "Perpetual" },
];

/** TermsFields are the license terms staff set, shared by issuing and editing. */
export const TermsFields = (): ReactElement => (
  <>
    <Form.TextField
      path="label"
      label="Label"
      inputProps={{ autoFocus: true, placeholder: "Test stand, site B" }}
    />
    <Form.Field<Term> path="term" label="Term">
      {(p) => (
        <Select.Static<Term, { key: Term; name: string }>
          {...p}
          data={TERMS}
          resourceName="Term"
        />
      )}
    </Form.Field>
    <Form.NumericField path="nodes" label="Machines that may activate" />
    <Form.NumericField
      path="channels"
      label="Channel cap per Core"
      inputProps={{ placeholder: "0 for unlimited" }}
    />
    <Form.Field<string>
      path="expiresAt"
      label="Expires (UTC)"
      visible={(_, ctx) => ctx.get<Term>("term")?.value === "subscription"}
    >
      {(p) => <Input.Text {...p} type="date" />}
    </Form.Field>
    <Form.TextField
      path="maxVersion"
      label="Maximum Core version"
      inputProps={{ placeholder: "0.62" }}
    />
    <Text.Text level="small" color={9}>
      Required on a perpetual license. On a subscription it is the version the license
      keeps covering after it expires.
    </Text.Text>
  </>
);
