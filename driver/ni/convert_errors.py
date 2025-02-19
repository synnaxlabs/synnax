#!/usr/bin/env python3

import re

def convert_file(input_path):
    with open(input_path, 'r') as f:
        content = f.read()
    
    # Regular expression to match the error entries
    pattern = r'\{\s*([^,]+),\s*([^,]+),\s*"([^"]+)"\s*\}'
    
    # Replace with designated initializer syntax
    def replacement(match):
        category, action, desc = match.groups()
        return (f'{{\n            .category={category.strip()}, '
                f'.actionRequired={action.strip()}, '
                f'.description="{desc.strip()}"\n        }}')
    
    modified = re.sub(pattern, replacement, content)
    
    with open(input_path, 'w') as f:
        f.write(modified)

if __name__ == '__main__':
    convert_file('error.h')