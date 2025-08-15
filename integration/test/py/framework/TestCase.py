#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from dataclasses import dataclass, field
from enum import Enum, auto
import random
from abc import ABC, abstractmethod
from typing import Any, Optional


@dataclass
class SynnaxConnection:
    """Data class representing the Synnax connection parameters."""
    server_address: str
    port: int
    username: str
    password: str
    secure: bool

class TestStatus(Enum):
    """Enum representing the status of a test."""
    INITIALIZING = auto()
    RUNNING = auto()
    COMPLETED = auto()
    PENDING = auto()
    FAILED = auto()
    KILLED = auto()
    TIMEOUT = auto()


class TestCase:
    """
    Parent class for all test cases in the integration test framework.
    
    This class handles the connection to the Synnax server and provides
    three key lifecycle methods that can be overridden by subclasses:
    - setup(): Called before the test runs
    - run(): The main test logic (must be implemented by subclasses)
    - teardown(): Called after the test runs
    """
    
    def __init__(self, SynnaxConnection: SynnaxConnection):
        """
        Initialize the test case with connection to Synnax server.
        
        Args:
            SynnaxConnection: The connection parameters for the Synnax server
        """
        # Generate a 6-character random alphanumeric string
        self.STATUS = TestStatus.INITIALIZING

        # Expected timeout in seconds (-1 means no timeout specified)
        self.Expected_Timeout: int = -1

        # Generate name
        self.name = self.__class__.__name__.lower()

        # Connect to Synnax server
        self.client = sy.Synnax(
            host=SynnaxConnection.server_address,
            port=SynnaxConnection.port,
            username=SynnaxConnection.username,
            password=SynnaxConnection.password,
            secure=SynnaxConnection.secure,
        )
        
        # Default rate
        self.loop = sy.Loop(1)
        
        """
        Define Test case channels
        """
        self.time_index = self.client.channels.create(
            name=f"{self.name}_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )  
        
        self.tlm = {
            f"{self.name}_time": sy.TimeStamp.now(),
        }
        
        self.add_channel(name="_uptime", data_type=sy.DataType.UINT32, initial_value=0)
        self.add_channel(name="_state", data_type=sy.DataType.UINT8, initial_value=self.STATUS.value)



    def add_channel(self, name: str, data_type: sy.DataType, initial_value: Any = None):

        """
        This function Exists for your convenience.
        It will create a channel with the name {self.name}_{name}
        and the data type {data_type}
        """

        self.client.channels.create(
            name=f"{self.name}_{name}",
            data_type=data_type,
            index=self.time_index.key,
            retrieve_if_name_exists=True,
        )
        self.tlm[f"{self.name}_{name}"] = initial_value


    def setup(self) -> None:
        """
        Setup logic. Take inputs, create channels, etc.
        Override this method in subclasses to implement test-specific setup logic.
        """


        pass
    
    @abstractmethod
    def run(self) -> None:
        """
        Main test logic method.
        This method must be implemented by all subclasses.
        """
        pass
    
    def teardown(self) -> None:
        """
        Teardown method called after the test runs.
        Override this method in subclasses to implement test-specific cleanup logic.
        """
        pass
    
    def execute(self) -> None:
        """
        Execute the complete test lifecycle: setup -> run -> teardown.
        This method should typically be called to run the test.
        """
        try:
            self.setup()
            self.run()
        finally:
            self.teardown()
