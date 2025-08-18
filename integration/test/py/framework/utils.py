#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import re

def validate_and_sanitize_name(name: str) -> str:
        """Sanitize name to contain only alphanumeric characters, hyphens, and underscores."""
        sanitized = re.sub(r'[^a-zA-Z0-9_-]', '', name)
        
        if not sanitized:
            raise ValueError("Name must contain at least one alphanumeric character")
        
        sanitized = sanitized.strip('_-')
        if not sanitized:
            raise ValueError("Name cannot consist only of hyphens and underscores")
            
        return sanitized