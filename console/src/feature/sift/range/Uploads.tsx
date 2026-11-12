// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/sift/range/Uploads.css";

import { type ranger, sift, type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Device, Ranger, Status as PStatus } from "@synnaxlabs/pluto";
import { bounds, TimeRange } from "@synnaxlabs/x";

import { MAKE } from "@/feature/sift/device/types";
import { useList as useUploadKeys } from "@/feature/sift/range/list";
import { useUploadModal } from "@/feature/sift/range/useUploadModal";
import { CSS } from "@/platform/css";

const { useResult: useUpload } = PStatus.createRetrieve(sift.uploadDetailsZ);

/**
 * @returns the fraction of the upload's time range, from 0 to 1, before which Sift has
 * confirmed every sample.
 */
const progress = ({ timeRange, positions }: sift.UploadDetails): number => {
  if (positions.length === 0) return 0;
  const tr = new TimeRange(timeRange);
  const reached = positions.reduce(
    (min, p) => (p.time.before(min) ? p.time : min),
    positions[0].time,
  );
  const done = new TimeRange(tr.start, reached).span.valueOf();
  return bounds.clamp({ lower: 0, upper: 1 }, Number(done) / Number(tr.span.valueOf()));
};

const UploadListItem = (props: List.ItemProps<status.Key>) => {
  const upload = useUpload({ key: props.itemKey }).data;
  const { update: del } = PStatus.useDelete();
  if (upload == null) return null;
  const { variant, message, description } = upload;
  // The status cache holds the details as they arrived, without the schema's types.
  const details = sift.uploadDetailsZ.parse(upload.details);
  const pending = variant === "loading" || variant === "warning";
  return (
    <List.Item {...props} justify="between" align="center">
      <Flex.Box y gap="tiny">
        <Flex.Box x align="center" gap="small">
          <Status.Indicator variant={variant} />
          <Text.Text weight={450}>{details.asset}</Text.Text>
          <Text.Text color={9}>{message}</Text.Text>
          {pending && (
            <Text.Text color={9} className={CSS.BE("sift-uploads", "progress")}>
              {Math.floor(progress(details) * 100)}%
            </Text.Text>
          )}
        </Flex.Box>
        {description != null && description !== "" && (
          <Text.Text level="small" color={9}>
            {description}
          </Text.Text>
        )}
      </Flex.Box>
      <Button.Button
        variant="text"
        tooltip={pending ? "Cancel upload" : "Delete upload"}
        aria-label={pending ? "Cancel upload" : "Delete upload"}
        onClick={() => del(upload.key)}
      >
        <Icon.Close />
      </Button.Button>
    </List.Item>
  );
};

const uploadListItem = Component.renderProp(UploadListItem);

const SIFT_DEVICE_QUERY = { where: { make: { eq: MAKE } }, limit: 1 };

export interface UploadsProps {
  rangeKey: ranger.Key;
}

/**
 * The Sift uploads of a range, with their progress, and an action that uploads the
 * range. It renders nothing until the range has an upload or a Sift device exists, or
 * the uploads fail to list.
 */
export const Uploads = ({ rangeKey }: UploadsProps) => {
  const rangeName = Ranger.useResult({ key: rangeKey }).data?.name ?? "";
  const devices = Device.useResultMultiple(SIFT_DEVICE_QUERY).data ?? [];
  const open = useUploadModal();
  const uploads = useUploadKeys(rangeKey);
  const data = uploads.data ?? [];
  const failed = uploads.variant === "error";
  if (!failed && data.length === 0 && devices.length === 0) return null;
  return (
    <Flex.Box y>
      <Header.Header level="h4" bordered borderColor={6}>
        <Header.Title color={11} weight={450}>
          Sift uploads
        </Header.Title>
        <Header.Actions>
          <Button.Button
            size="medium"
            variant="text"
            tooltip="Upload to Sift"
            aria-label="Upload to Sift"
            onClick={() => open({ rangeKey, rangeName })}
          >
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      {failed ? (
        <Status.Summary status={uploads.status} />
      ) : (
        <List.Frame data={data}>
          <List.Scroll>
            <List.Items>{uploadListItem}</List.Items>
          </List.Scroll>
        </List.Frame>
      )}
    </Flex.Box>
  );
};
