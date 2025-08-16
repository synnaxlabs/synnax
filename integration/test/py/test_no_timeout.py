#!/usr/bin/env python3
"""
Test script to demonstrate that when timeout=-1, timeout checks are ignored.

This script creates a test case with timeout=-1 and verifies that
the test conductor does not monitor it for timeouts.
"""

import time
import sys
import os

# Add the framework directory to the path
sys.path.append(os.path.join(os.path.dirname(__file__), 'framework'))

from framework.TestCase import TestCase, STATUS
from framework.Test_Conducter import Test_Conductor, TestDefinition

class NoTimeoutTest(TestCase):
    """Test case with no timeout (Expected_Timeout=-1) to test timeout ignoring."""
    
    def __init__(self, SynnaxConnection):
        super().__init__(SynnaxConnection)
        # Set no timeout (-1) to test that timeout checks are ignored
        self.Expected_Timeout = -1
        
    def setup(self):
        """Setup the test."""
        super().setup()
        print(f"{self.name} > Setup complete")
        
    def run(self):
        """Main test logic that should not be killed by timeout."""
        print(f"{self.name} > Starting test execution...")
        print(f"{self.name} > This test has Expected_Timeout=-1, so it should not be killed by timeout")
        
        # Simulate work that would normally exceed a timeout
        for i in range(3):
            print(f"{self.name} > Working... {i+1}/3")
            time.sleep(1)
            
        print(f"{self.name} > Test execution complete (this should be reached since no timeout)")
        
    def teardown(self):
        """Teardown the test."""
        print(f"{self.name} > Teardown called")
        super().teardown()

def test_no_timeout():
    """Test that the test conductor ignores timeout checks when Expected_Timeout=-1."""
    print("Testing no timeout functionality (Expected_Timeout=-1)...")
    
    # Create test conductor
    conductor = Test_Conductor(name="no_timeout_test_conductor")
    
    # Create a test definition for our no-timeout test
    test_def = TestDefinition(case="test_no_timeout.NoTimeoutTest")
    
    # Load the test sequence
    conductor.test_definitions = [test_def]
    
    try:
        # Run the test sequence
        print("Starting test execution...")
        results = conductor.run_sequence()
        
        # Check the results
        if results:
            result = results[0]
            print(f"\nTest result: {result.status.name}")
            if result.error_message:
                print(f"Error message: {result.error_message}")
            
            # Verify that the test completed successfully (not killed)
            if result.status == STATUS.PASSED:
                print("✓ SUCCESS: Test completed successfully without being killed by timeout")
                return True
            else:
                print(f"✗ FAILURE: Expected test to pass, but got status: {result.status.name}")
                return False
        else:
            print("✗ FAILURE: No test results returned")
            return False
            
    except Exception as e:
        print(f"✗ ERROR: {e}")
        return False
    finally:
        # Clean up
        conductor.shutdown()

if __name__ == "__main__":
    success = test_no_timeout()
    sys.exit(0 if success else 1)
