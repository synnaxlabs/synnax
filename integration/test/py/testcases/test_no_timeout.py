#!/usr/bin/env python3
"""
Test script to demonstrate that when timeout=-1, timeout checks are ignored.

This script creates a test case with timeout=-1 and verifies that
the test conductor does not monitor it for timeouts.
"""

import os
import sys
import time

from framework.test_case import STATUS, TestCase
from framework.test_conductor import TestConductor, TestDefinition


class NoTimeoutTest(TestCase):
    """Test case with no timeout (Expected_Timeout=-1) to test timeout ignoring."""

    def setup(self):
        """Setup the test."""

        # Set no timeout (-1) to test that timeout checks are ignored
        self.Expected_Timeout = -1
        super().setup()
        self._log_message("Setup complete")

    def run(self):
        """Main test logic that should not be killed by timeout."""
        self._log_message("Starting test execution...")
        self._log_message(
            "This test has Expected_Timeout=-1, so it should not be killed by timeout"
        )

        # Simulate work that would normally exceed a timeout
        for i in range(3):
            self._log_message(f"Working... {i+1}/3")
            time.sleep(1)

        self._log_message(
            "Test execution complete (this should be reached since no timeout)"
        )

    def teardown(self):
        """Teardown the test."""
        self._log_message("Teardown called")
        super().teardown()


def test_no_timeout(self):
    """Test that the test conductor ignores timeout checks when Expected_Timeout=-1."""
    self._log_message("Testing no timeout functionality (Expected_Timeout=-1)...")

    # Create test conductor
    conductor = TestConductor(name="no_timeout_test_conductor")

    # Create a test definition for our no-timeout test
    test_def = TestDefinition(case="test_no_timeout.NoTimeoutTest")

    # Load the test sequence
    conductor.test_definitions = [test_def]

    try:
        # Run the test sequence
        self._log_message("Starting test execution...")
        results = conductor.run_sequence()

        # Check the results
        if results:
            result = results[0]
            self._log_message(f"Test result: {result.status.name}")
            if result.error_message:
                self._log_message(f"Error message: {result.error_message}")

            # Verify that the test completed successfully (not killed)
            if result.status == STATUS.PASSED:
                self._log_message(
                    "✓ SUCCESS: Test completed successfully without being killed by timeout"
                )
                return True
            else:
                self._log_message(
                    f"✗ FAILURE: Expected test to pass, but got status: {result.status.name}"
                )
                return False
        else:
            self._log_message("✗ FAILURE: No test results returned")
            return False

    except Exception as e:
        self._log_message(f"✗ ERROR: {e}")
        return False
    finally:
        # Clean up
        conductor.shutdown()
