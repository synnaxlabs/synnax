#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import importlib
import json
from pathlib import Path
from typing import Any

import pytest
from pydantic import BaseModel, TypeAdapter

import synnax
from synnax.device import actions_gen as device_actions
from synnax.ranger import actions_gen as ranger_actions
from x.codec import JSONCodec

CASES = [
    pytest.param(
        ranger_actions.Action,
        ranger_actions.set(
            ranger_actions.SetPayload(path=["time_range", "end"], value=100)
        ),
        {
            "type": "set",
            "set": {"path": ["time_range", "end"], "value": 100, "guards": []},
        },
        id="range",
    ),
    pytest.param(
        device_actions.Action,
        device_actions.set(
            device_actions.SetPayload(path=["properties"], value={"port": 502})
        ),
        {
            "type": "set",
            "set": {"path": ["properties"], "value": {"port": 502}, "guards": []},
        },
        id="device",
    ),
    pytest.param(
        ranger_actions.Action,
        ranger_actions.set(
            ranger_actions.SetPayload(path=["time_range", "end"], value=None)
        ),
        {"type": "set", "set": {"path": ["time_range", "end"], "guards": []}},
        id="null",
    ),
]

ROOT = Path(synnax.__file__).parent
MODULES = sorted(
    ".".join(("synnax", *p.relative_to(ROOT).with_suffix("").parts))
    for p in ROOT.rglob("actions_gen.py")
)


class TestActions:
    """Tests for the action modules that Oracle generates."""

    @pytest.mark.parametrize(("action_t", "action", "envelope"), CASES)
    def test_encode(
        self, action_t: Any, action: BaseModel, envelope: dict[str, Any]
    ) -> None:
        """Should encode an action as the JSON envelope that the Core reads."""
        assert json.loads(JSONCodec().encode(action)) == envelope

    @pytest.mark.parametrize(("action_t", "action", "envelope"), CASES)
    def test_decode(
        self, action_t: Any, action: BaseModel, envelope: dict[str, Any]
    ) -> None:
        """Should decode an action from its JSON envelope."""
        assert TypeAdapter(action_t).validate_python(envelope) == action

    @pytest.mark.parametrize("module", MODULES)
    def test_import(self, module: str) -> None:
        """Should import every action module and build its Action adapter."""
        TypeAdapter(importlib.import_module(module).Action)
