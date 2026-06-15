// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/user/Badge.css";

import { Button, Dialog, Icon, Status, User } from "@synnaxlabs/pluto";
import { type ChangeEvent, type ReactElement, useCallback, useRef } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { logout } from "@/cluster/services/logout";
import { fileToAvatarDataURI } from "@/user/image";

export const Badge = (): ReactElement | null => {
  const dispatch = useDispatch();
  const { data: u } = User.useRetrieve({}, { addStatusOnFailure: false });
  const cluster = Cluster.useSelect();
  const handleError = Status.useErrorHandler();
  const { updateAsync: changeAvatar } = User.useChangeAvatar();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const handleLogout = useCallback(() => logout(dispatch), [dispatch]);
  const username = u?.username ?? cluster?.username ?? "";
  const displayName =
    u?.firstName != null && u?.firstName != "" ? u.firstName : username;
  const handleFileChange = (e: ChangeEvent<HTMLInputElement>): void => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (file == null || u == null) return;
    handleError(async () => {
      const avatar = await fileToAvatarDataURI(file);
      await changeAvatar({ key: u.key, avatar });
    }, "Failed to update avatar");
  };
  const handleRemoveAvatar = (): void => {
    if (u == null) return;
    handleError(
      async () => await changeAvatar({ key: u.key, avatar: "" }),
      "Failed to remove avatar",
    );
  };
  const hasAvatar = u?.avatar != null && u.avatar !== "";
  return (
    <Dialog.Frame>
      <Dialog.Trigger contrast={2} hideCaret textColor={10} gap="small" weight={400}>
        <User.Avatar username={username} src={u?.avatar} />
        {displayName}
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered
        borderColor={6}
        style={{ padding: "1rem", width: 200 }}
        gap="small"
      >
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          style={{ display: "none" }}
          onChange={handleFileChange}
        />
        <Button.Button
          onClick={() => fileInputRef.current?.click()}
          variant="text"
          full="x"
          disabled={u == null}
        >
          <Icon.Edit />
          {hasAvatar ? "Change photo" : "Upload photo"}
        </Button.Button>
        {hasAvatar && (
          <Button.Button onClick={handleRemoveAvatar} variant="text" full="x">
            <Icon.Delete />
            Remove photo
          </Button.Button>
        )}
        <Button.Button onClick={handleLogout} variant="text" full="x">
          <Icon.Logout />
          Log out
        </Button.Button>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
