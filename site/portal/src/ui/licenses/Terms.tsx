// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useId } from "react";
import { z } from "zod";

import { type License, type Term } from "@/server/db/schema";
import { VERSION_PATTERN } from "@/server/license/limits";

const termChoiceZ = z.enum(["subscription", "updates", "perpetual"]);

/**
 * TermChoice is the term staff pick. "updates" saves as a subscription with a maximum
 * version.
 */
type TermChoice = z.infer<typeof termChoiceZ>;

/** termsSchema is the unrefined shape; apply checkTerms to get the term rules. */
export const termsSchema = z.object({
  label: z.string().trim().min(1, "Give the license a label"),
  term: termChoiceZ,
  nodes: z.number().int().min(1, "At least one machine"),
  channelsLimited: z.boolean(),
  channels: z.number(),
  expiresAt: z.string(),
  maxVersion: z.string().trim(),
});

export interface TermsValues extends z.infer<typeof termsSchema> {}

/** checkTerms holds the rules that tie the term to the expiry and version ceiling. */
export const checkTerms = (ctx: z.core.ParsePayload<TermsValues>): void => {
  const v = ctx.value;
  if (v.term !== "perpetual" && v.expiresAt === "")
    ctx.issues.push({
      code: "custom",
      message:
        v.term === "updates"
          ? "Enter the date updates end"
          : "A subscription needs an expiry date",
      path: ["expiresAt"],
      input: v.expiresAt,
    });
  if (v.channelsLimited && !Number.isInteger(v.channels))
    ctx.issues.push({
      code: "custom",
      message: "Use a whole number",
      path: ["channels"],
      input: v.channels,
    });
  else if (v.channelsLimited && v.channels < 1)
    ctx.issues.push({
      code: "custom",
      message: "At least one channel",
      path: ["channels"],
      input: v.channels,
    });
  if (v.term === "subscription") return;
  if (v.maxVersion === "")
    ctx.issues.push({
      code: "custom",
      message: "A perpetual license needs a maximum version",
      path: ["maxVersion"],
      input: v.maxVersion,
    });
  else if (!VERSION_PATTERN.test(v.maxVersion))
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
  channelsLimited: false,
  channels: 0,
  expiresAt: "",
  maxVersion: "",
};

/** termsOf reads a license into the values the form edits. */
export const termsOf = (lic: License): TermsValues => ({
  label: lic.label,
  term: lic.term === "subscription" && lic.maxVersion != null ? "updates" : lic.term,
  nodes: lic.nodes,
  channelsLimited: lic.channels > 0,
  channels: lic.channels,
  expiresAt: lic.expiresAt?.toISOString().slice(0, 10) ?? "",
  maxVersion: lic.maxVersion ?? "",
});

/**
 * bodyOf reads form values into the body the license routes take, where a channel cap
 * of 0 is no cap and an empty maximum version is no ceiling. A subscription never
 * carries a maximum version.
 */
export const bodyOf = <V extends TermsValues>({
  term,
  channelsLimited,
  ...rest
}: V): Omit<V, "term" | "channelsLimited"> & { term: Term } => ({
  ...rest,
  term: term === "perpetual" ? "perpetual" : "subscription",
  channels: channelsLimited ? rest.channels : 0,
  maxVersion: term === "subscription" ? "" : rest.maxVersion,
});

const TERM_DESCRIPTIONS: Record<TermChoice, string> = {
  subscription: "Covers every Core version until it expires.",
  updates:
    "Never expires. Covers every Core version until updates end, then Core versions " +
    "up to the maximum version.",
  perpetual: "Never expires. Covers Core versions up to the maximum version.",
};

/** TermsFields are the license terms staff set, shared by issuing and editing. */
export const TermsFields = (): ReactElement => {
  const term = Form.useFieldValue<TermChoice>("term");
  const channelsLimited = Form.useField<boolean>("channelsLimited");
  const channelsLimitedID = useId();
  return (
    <>
      <Form.TextField
        path="label"
        label="Label"
        inputProps={{ autoFocus: true, placeholder: "Test stand, site B" }}
      />
      <Flex.Box y gap="small">
        <Form.Field<TermChoice> path="term" label="Term" padHelpText={false}>
          {(p) => (
            <Select.Simple<TermChoice> {...p} resourceName="Term">
              <Select.Item itemKey="subscription">Subscription</Select.Item>
              <Select.Item itemKey="updates">Perpetual with updates</Select.Item>
              <Select.Item itemKey="perpetual">Perpetual</Select.Item>
            </Select.Simple>
          )}
        </Form.Field>
        <Text.Text level="small" color={9}>
          {TERM_DESCRIPTIONS[term]}
        </Text.Text>
      </Flex.Box>
      <Form.Field<string>
        path="expiresAt"
        label={term === "updates" ? "Updates until (UTC)" : "Expires (UTC)"}
        visible={term !== "perpetual"}
      >
        {(p) => <Input.Text {...p} type="date" />}
      </Form.Field>
      <Form.TextField
        path="maxVersion"
        label="Maximum version"
        visible={term !== "subscription"}
        inputProps={{ placeholder: "0.62" }}
      />
      <Form.NumericField path="nodes" label="Machines that may activate" />
      <Flex.Box x align="center" gap="small">
        <Input.Checkbox
          id={channelsLimitedID}
          value={channelsLimited.value}
          onChange={channelsLimited.onChange}
          size="small"
        />
        <Text.Text el="label" htmlFor={channelsLimitedID} level="small" color={10}>
          Limit channels per Core
        </Text.Text>
      </Flex.Box>
      <Form.NumericField
        path="channels"
        label="Channels per Core"
        visible={channelsLimited.value}
        inputProps={{ emptyValue: 0 }}
      />
    </>
  );
};
