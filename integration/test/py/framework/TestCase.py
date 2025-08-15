#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from abc import ABC, abstractmethod
from typing import Optional


class TestCase(ABC):
    """
    Parent class for all test cases in the integration test framework.
    
    This class handles the connection to the Synnax server and provides
    three key lifecycle methods that can be overridden by subclasses:
    - setup(): Called before the test runs
    - run(): The main test logic (must be implemented by subclasses)
    - teardown(): Called after the test runs
    """
    
    def __init__(self, server_address: str, port: int = 9090, 
                 username: str = "synnax", password: str = "seldon", 
                 secure: bool = False):
        """
        Initialize the test case with connection to Synnax server.
        
        Args:
            server_address: The address of the Synnax server (e.g., "localhost")
            port: The port number (default: 9090)
            username: Username for authentication (default: "synnax")
            password: Password for authentication (default: "seldon")
            secure: Whether to use secure connection (default: False)
        """
        self.server_address = server_address
        self.port = port
        self.username = username
        self.password = password
        self.secure = secure
        
        # Connect to Synnax server
        self.client = sy.Synnax(
            host=server_address,
            port=port,
            username=username,
            password=password,
            secure=secure,
        )
        
        # Expected timeout in seconds (-1 means no timeout specified)
        self.Expected_Timeout: int = -1
    
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
