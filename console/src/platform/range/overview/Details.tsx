// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/overview/Details.css";

import { type label, ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Label as PLabel, Ranger } from "@synnaxlabs/pluto";
import { type NumericTimeRange } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Framer } from "@/platform/framer";
import { Label } from "@/platform/label";
import { Panel } from "@/platform/panel";
import { FavoriteButton } from "@/platform/range/FavoriteButton";

interface ParentRangeButtonProps {
  rangeKey: string;
}

const Internal = ({ rangeKey }: ParentRangeButtonProps): ReactElement | null => {
  const parent = Ranger.useParent({ id: ranger.ontologyID(rangeKey) });
  const openTab = Panel.useOpenTab();
  if (parent == null) return null;
  const Icon = Ranger.STAGE_ICONS[Ranger.getStage(parent.timeRange)];
  return (
    <Flex.Box x gap="small" align="center">
      <Text.Text weight={450} color={9}>
        Child range of
      </Text.Text>
      <Button.Button
        color={9}
        variant="text"
        gap="small"
        className={CSS.BE("range-overview", "parent-button")}
        onClick={() =>
          openTab({ variant: "resource", resource: ranger.ontologyID(parent.key) })
        }
      >
        <Icon />
        {parent.name}
      </Button.Button>
    </Flex.Box>
  );
};

const ParentRangeButton = (props: ParentRangeButtonProps): ReactElement => (
  <Errors.SuspenseBoundary loading={null}>
    <Internal {...props} />
  </Errors.SuspenseBoundary>
);

interface LabelsProps {
  rangeKey: string;
  preview: boolean;
}

const Labels = ({ rangeKey, preview }: LabelsProps): ReactElement => {
  const value = (Ranger.useLabels(rangeKey) ?? []).map(({ key }) => key);
  const { update } = PLabel.useEditFor();
  const handleChange = (next: label.Key[]): void =>
    update({
      id: ranger.ontologyID(rangeKey),
      added: next.filter((key) => !value.includes(key)),
      removed: value.filter((key) => !next.includes(key)),
    });
  return (
    <Label.SelectMultiple
      zIndex={100}
      variant="floating"
      className={CSS.BE("range-overview", "labels-select")}
      value={value}
      onChange={handleChange}
      preview={preview}
    />
  );
};

export interface DetailsProps {
  rangeKey: string;
}

export const Details: FC<DetailsProps> = ({ rangeKey }) => {
  const range = Ranger.use({ key: rangeKey });
  // The form saves on every edit, so a subject who cannot write the range gets it
  // read-only rather than a field that reverts once the save is refused.
  const canEdit = Access.useUpdateGranted(ranger.ontologyID(rangeKey));
  const parent = Ranger.useResultParent({ id: ranger.ontologyID(rangeKey) }).data;
  const { form } = Ranger.useForm({
    query: { key: rangeKey },
    mode: canEdit ? "normal" : "preview",
  });

  const handleError = Status.useErrorHandler();
  const name = Form.useFieldValue<string, string, typeof Ranger.formSchema>("name", {
    ctx: form,
  });

  const getPythonCode = () =>
    `
      # Retrieve ${name}
      rng = client.ranges.retrieve(key="${rangeKey}")
    `;
  const getTypeScriptCode = () =>
    `
      // Retrieve ${name}
      const rng = await client.ranges.retrieve("${rangeKey}")
    `;

  const promptDownloadCSVModal = Framer.useDownloadCSVModal();

  return (
    <Form.Form<typeof Ranger.formSchema> {...form}>
      <Flex.Box y gap="large">
        <Flex.Box x justify="between" className={CSS.B("header")}>
          <Flex.Box y grow>
            <Form.TextField
              path="name"
              showLabel={false}
              inputProps={{
                variant: "text",
                level: "h1",
                placeholder: "Name",
                onlyChangeOnBlur: true,
                resetOnBlurIfEmpty: true,
              }}
              padHelpText={false}
            />
            <ParentRangeButton rangeKey={rangeKey} />
          </Flex.Box>
          <Flex.Box x className={CSS.BE("range-overview", "actions")} gap="small">
            <Button.Copy
              text={getPythonCode}
              tooltip={`Copy Python code for ${name}`}
              tooltipLocation="bottom"
              variant="text"
              successMessage={{ message: "Copied Python code to clipboard", name }}
              textColor={9}
            >
              <Icon.Python />
            </Button.Copy>
            <Button.Copy
              variant="text"
              text={getTypeScriptCode}
              tooltip={`Copy TypeScript code for ${name}`}
              tooltipLocation="bottom"
              successMessage={{ message: "Copied TypeScript code to clipboard", name }}
              textColor={9}
            >
              <Icon.TypeScript />
            </Button.Copy>
            <Divider.Divider y />
            <Button.Button
              tooltip={`Download data for ${name} as CSV`}
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
              onClick={() =>
                handleError(async () => {
                  await promptDownloadCSVModal({
                    timeRange: form.get<NumericTimeRange>("timeRange").value,
                    name,
                    channels: [],
                    icon: <Icon.Range />,
                  });
                }, "Failed to download CSV")
              }
            >
              <Icon.CSV />
            </Button.Button>
            <Divider.Divider y />
            <Core.CopyLinkToolbarButton
              name={name}
              ontologyID={ranger.ontologyID(rangeKey)}
              tooltip={`Copy link to ${name}`}
              tooltipLocation="bottom"
              textColor={9}
            />
            <FavoriteButton range={range} size="medium" />
          </Flex.Box>
        </Flex.Box>
        <Form.Field<NumericTimeRange>
          path="timeRange"
          padHelpText={false}
          showLabel={false}
        >
          {(p) => (
            <Ranger.Timeline
              level="h4"
              variant="shadow"
              parent={parent?.timeRange.numeric}
              {...p}
            />
          )}
        </Form.Field>
        <Flex.Box x>
          <Labels rangeKey={rangeKey} preview={!canEdit} />
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
