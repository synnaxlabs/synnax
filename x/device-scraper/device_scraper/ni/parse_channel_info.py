#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

success_count = 0
error_count = 0


def parse_channel_dio(i, product: dict):
    global success_count, error_count
    pinouts = product.get("pinouts", [])
    product["channels"] = dict()
    ports_candidates = list()
    lines_candidates = list()
    total_candidates = list()
    for pout in pinouts:
        dio = pout.get("info", {}).get("digitalIO", None)
        if dio is None:
            continue

        ports = dio.get("ports", None)
        if ports is not None:
            ports_candidates.append(ports)

        lines = dio.get("lines", None)
        if lines is not None:
            lines_candidates.append(lines)

    bidir_channels = product["productData"].get("NumberOfBidirectionalChannels", None)
    if bidir_channels is not None:
        total_candidates.append(bidir_channels)

    if (
        len(ports_candidates) == 0
        and len(lines_candidates) == 0
        and len(total_candidates) == 0
    ):
        return

    port_lower_candidates = set([x["lower"] for x in ports_candidates])
    power_upper_candidates = set([x["upper"] for x in ports_candidates])

    line_lower_candidates = set([x["lower"] for x in lines_candidates])
    line_upper_candidates = set([x["upper"] for x in lines_candidates])

    # For upper, get teh largest value
    port_upper = max(power_upper_candidates, default=0)
    port_lower = min(port_lower_candidates, default=0)
    line_upper = max(line_upper_candidates, default=0)
    line_lower = min(line_lower_candidates, default=0)

    # port
    port_count = port_upper - port_lower + 1
    line_count = line_upper - line_lower + 1
    total = max(total_candidates, default=0)
    if int(total) != 0 and (int(port_count) * int(line_count)) != int(total):
        print(
            f"Error: {i} {product['tileLabel']}: ({port_count} * {line_count} != {total})"
        )
        error_count += 1
    else:
        success_count += 1

    product["channels"]["digitalInputOutput"] = {
        "ports": {"upper": port_upper, "lower": port_lower},
        "lines": {"upper": line_upper, "lower": line_lower},
    }


input_products = json.load(open("out/products.json"))

for i, product in enumerate(input_products):
    parse_channel_dio(i, product)

print(f"Success: {success_count}, Error: {error_count}")
json.dump(input_products, open("out/products-2.json", "w"), indent=2)
