#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

from synnax import panel


class TestPanel:
    def test_defaults(self) -> None:
        """Should create a panel with an empty leaf from only a name."""
        p = panel.Panel(name="Ops")
        assert p.root == panel.LeafNode(tabs=[])

    def test_round_trip(self) -> None:
        """Should parse a split tree of tabs and dump it back unchanged."""
        p = panel.Panel.model_validate(
            {
                "name": "Ops",
                "root": {
                    "variant": "split",
                    "direction": "x",
                    "size": 0.5,
                    "first": {
                        "variant": "leaf",
                        "tabs": [
                            {
                                "variant": "resource",
                                "resource": {"type": "lineplot", "key": "k"},
                            }
                        ],
                    },
                    "last": {
                        "variant": "leaf",
                        "tabs": [{"variant": "view", "type": "docs"}],
                    },
                },
            }
        )
        assert isinstance(p.root, panel.SplitNode)
        assert isinstance(p.root.first, panel.LeafNode)
        assert isinstance(p.root.first.tabs[0], panel.ResourceTab)
        assert isinstance(p.root.last, panel.LeafNode)
        assert isinstance(p.root.last.tabs[0], panel.ViewTab)
        assert panel.Panel.model_validate_json(p.model_dump_json()) == p

    def test_ontology_id(self) -> None:
        """Should build the ontology ID of a panel."""
        key = uuid4()
        assert panel.ontology_id(key).type == panel.ONTOLOGY_TYPE.type
        assert panel.ontology_id(key).key == str(key)
        p = panel.Panel(name="Ops")
        assert hash(p) == hash(p.key)
