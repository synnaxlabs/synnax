#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Page

from console.context_menu import ContextMenu
from console.layout import LayoutClient
from console.notifications import NotificationsClient
from console.tree import Tree


class ResourceClient:
    """Shared wiring for resource helper clients.

    Exposes the one Page, ContextMenu, NotificationsClient, and Tree the
    injected LayoutClient owns, so every client shares the same instances.
    """

    layout: LayoutClient
    page: Page
    ctx_menu: ContextMenu
    notifications: NotificationsClient
    tree: Tree

    def __init__(self, layout: LayoutClient) -> None:
        self.layout = layout
        self.page = layout.page
        self.ctx_menu = layout.ctx_menu
        self.notifications = layout.notifications
        self.tree = layout.tree
