#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
import re


# Also suppress stderr for WebSocket errors
class WebSocketErrorFilter:
    def __init__(self):
        self.original_stderr = sys.stderr
        
    def write(self, text):
        if any(phrase in text for phrase in [
            "keepalive ping", "1011", "timed out while closing connection",
            "ConnectionClosedError", "WebSocketException"
        ]):
            return
        self.original_stderr.write(text)
        
    def flush(self):
        self.original_stderr.flush()

# More aggressive WebSocket error suppression
def ignore_websocket_errors(type, value, traceback):
    error_str = str(value)
    if any(phrase in error_str for phrase in [
        "keepalive ping", "1011", "timed out while closing connection", 
        "ConnectionClosedError", "WebSocketException"
    ]):
        return
    sys.__excepthook__(type, value, traceback)


def validate_and_sanitize_name(name: str) -> str:
        """Sanitize name to contain only alphanumeric characters, hyphens, and underscores."""
        sanitized = re.sub(r'[^a-zA-Z0-9_-]', '', name)
        
        if not sanitized:
            raise ValueError("Name must contain at least one alphanumeric character")
        
        sanitized = sanitized.strip('_-')
        if not sanitized:
            raise ValueError("Name cannot consist only of hyphens and underscores")
            
        return sanitized
