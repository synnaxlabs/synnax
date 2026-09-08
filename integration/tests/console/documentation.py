#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console import docs as docs_client
from console.case import ConsoleCase


class Documentation(ConsoleCase):
    """The Console's documentation actions hand the docs URL to the browser."""

    def run(self) -> None:
        docs = self.console.docs

        self.log("(1/2) Open documentation from the command palette")
        tab = docs.open_via_command_palette()
        assert tab.url == docs_client.URL, f"Palette opened {tab.url}"
        tab.close()

        self.log("(2/2) Open documentation from the question mark icon")
        tab = docs.open_via_question_mark_icon()
        assert tab.url == docs_client.URL, f"Icon opened {tab.url}"
        tab.close()
