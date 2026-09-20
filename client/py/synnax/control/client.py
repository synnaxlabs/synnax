#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel, Field

from freighter import UnaryClient
from synnax import channel
from synnax.channel import Key
from synnax.control.controller import Controller
from synnax.framer import Client as FrameClient
from x.control import Authority, CrudeAuthority, State
from x.params import require_named_params

_RETRIEVE_ENDPOINT = "/control/retrieve"


class _RetrieveRequest(BaseModel):
    keys: list[Key] = Field(default_factory=list)


class _RetrieveResponse(BaseModel):
    states: list[State[Key]] = Field(default_factory=list)


class Client:
    """Central client for control related tasks in the Synnax client, primarily used for
    acquiring control to run sequences.
    """

    framer: FrameClient
    retriever: channel.Retriever
    client: UnaryClient

    def __init__(
        self,
        framer: FrameClient,
        channels: channel.Retriever,
        client: UnaryClient,
    ) -> None:
        self.framer = framer
        self.retriever = channels
        self.client = client

    @overload
    def retrieve(self, *, key: Key) -> State[Key] | None: ...

    @overload
    def retrieve(self, *, name: str) -> State[Key] | None: ...

    @overload
    def retrieve(self, *, keys: list[Key]) -> list[State[Key]]: ...

    @overload
    def retrieve(self, *, names: list[str]) -> list[State[Key]]: ...

    @overload
    def retrieve(self) -> list[State[Key]]: ...

    @require_named_params(example_params=("name", "'valve_cmd'"))
    def retrieve(
        self,
        *,
        key: Key | None = None,
        name: str | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
    ) -> State[Key] | list[State[Key]] | None:
        """Retrieves the current control state of channels in the cluster.

        :param key: Read the state of a single channel by key.
        :param name: Read the state of a single channel by name.
        :param keys: Read the state of several channels by key.
        :param names: Read the state of several channels by name.
        :returns: A single state, or None when the channel is uncontrolled, if key or
            name is given. Otherwise one state per controlled channel, with
            uncontrolled channels absent. Passing nothing reads every controlled
            channel in the cluster.
        :raises NotFoundError: If the channel given by key or name does not exist.
        """
        single = key if key is not None else name
        if single is not None:
            resolved = self.retriever.retrieve_one(single)
            states = self._exec_retrieve([resolved.key])
            return states[0] if len(states) > 0 else None
        if keys is None and names is not None:
            keys = [ch.key for ch in self.retriever.retrieve(names)]
        if keys is None:
            return self._exec_retrieve([])
        # An empty key set reads the whole cluster, so it must not reach the Core here.
        return self._exec_retrieve(keys) if len(keys) > 0 else []

    def _exec_retrieve(self, keys: list[Key]) -> list[State[Key]]:
        return self.client.send(
            _RETRIEVE_ENDPOINT, _RetrieveRequest(keys=keys), _RetrieveResponse
        ).states

    def acquire(
        self,
        name: str,
        read: channel.Params | None,
        write: channel.Params | None,
        write_authorities: CrudeAuthority | list[CrudeAuthority] = Authority.ABSOLUTE,
    ) -> Controller:
        """Opens a new controller for executing control sequences. We recommend using
        this method under a context manager to ensure that the controller is properly
        closed after use.

        :param name: A human-readable name for the controller to identify it across
        Synnax.
        :param read: A list of channels that the controller will need to read from in
        order to execute its sequences. Any channels not on this list will not be
        available to make control decisions.
        :param write: A list of channels that the controller will need to write to
        during operation. Any channels not on this list will not be writable by the
        controller.
        :param write_authorities: A single or list of authorities that the controller
        defines when writing to the channels. This can be a single authority or a list
        of authorities corresponding to the channels passed in the write parameter.
        """
        return Controller(
            name=name,
            write=write,
            read=read,
            frame_client=self.framer,
            retriever=self.retriever,
            write_authorities=write_authorities,
        )
