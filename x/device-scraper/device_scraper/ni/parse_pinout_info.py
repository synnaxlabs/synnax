#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from html.parser import HTMLParser
from paralell import process_products
import requests
import json
import warnings
import time


def add_at_index(items: list[set[int]], port: int, mbe_line_max: int = 1):
    if port >= len(items):
        items.extend([set()] * (port - len(items) + 1))
    items[port].add(mbe_line_max)


def estimate_pinout_accuracy(i, product):
    analog_input_count_opt: set[int] = set()
    analog_output_count_opt: set[int] = set()
    thermocouple_count_opt: set[int] = set()
    digital_input_count_opt: set[int] = set()
    digital_input_count_line_opt: list[set[int]] = list()
    digital_output_count_opt: set[int] = set()
    digital_output_count_line_opt: list[set[int]] = list()
    digital_io_count_opt: set[int] = set()
    digital_io_count_line_opt: list[set[int]] = list()

    pinouts = product.get("pinouts", [])
    for pinout in pinouts:
        if "info" not in pinout:
            continue
        info = pinout["info"]
        analog_input_count_opt.add(info["analogIn"]["portCount"])
        analog_output_count_opt.add(info["analogOut"]["portCount"])
        thermocouple_count_opt.add(info["thermocouple"]["portCount"])
        digital_input_count_opt.add(info["digitalInput"]["portCount"])
        digital_output_count_opt.add(info["digitalOutput"]["portCount"])
        digital_io_count_opt.add(info["digitalInputOutput"]["portCount"])

        for i in range(len(info["digitalInput"]["lineCounts"])):
            add_at_index(
                digital_input_count_line_opt, i, info["digitalInput"]["lineCounts"][i]
            )

        for i in range(len(info["digitalOutput"]["lineCounts"])):
            add_at_index(
                digital_output_count_line_opt, i, info["digitalOutput"]["lineCounts"][i]
            )

        for i in range(len(info["digitalInputOutput"]["lineCounts"])):
            add_at_index(
                digital_io_count_line_opt,
                i,
                info["digitalInputOutput"]["lineCounts"][i],
            )

    to_add_back = 0 if len(pinouts) == 0 else 1
    confidence = 100
    confidence -= len(analog_input_count_opt) - to_add_back
    max_analog_input = max(analog_input_count_opt)
    confidence -= len(analog_output_count_opt) - to_add_back
    max_analog_output = max(analog_output_count_opt)
    confidence -= len(thermocouple_count_opt) - to_add_back
    max_thermocouple = max(thermocouple_count_opt)
    confidence -= len(digital_input_count_opt) - to_add_back
    max_digital_input = max(digital_input_count_opt)
    confidence -= len(digital_output_count_opt) - to_add_back
    max_digital_output = max(digital_output_count_opt)
    confidence -= len(digital_io_count_opt) - to_add_back
    max_digital_io = max(digital_io_count_opt)
    confidence -= len(digital_input_count_line_opt) - to_add_back
    digital_io_line_maxes = list()
    digital_input_line_maxes = list()
    digital_output_line_maxes = list()

    for dio in range(max_digital_io):
        digital_io_line_maxes.append(max(digital_io_count_line_opt[dio]))
        confidence -= len(digital_io_count_line_opt[dio])

    for di in range(max_digital_input):
        digital_input_line_maxes.append(max(digital_input_count_line_opt[di]))
        confidence -= len(digital_input_count_line_opt[di])

    for do in range(max_digital_output):
        digital_output_line_maxes.append(max(digital_output_count_line_opt[do]))
        confidence -= len(digital_output_count_line_opt[do])

    out = {
        "confidence": confidence,
        "analogInput": {"portCount": max_analog_input},
        "analogOutput": {"portCount": max_analog_output},
        "thermocouple": {"portCount": max_thermocouple},
        "digitalInput": {
            "portCount": max_digital_input,
            "lineCounts": digital_input_line_maxes,
        },
        "digitalOutput": {
            "portCount": max_digital_output,
            "lineCounts": digital_output_line_maxes,
        },
        "digitalInputOutput": {
            "portCount": max_digital_io,
            "lineCounts": digital_io_line_maxes,
        },
    }
    product["estimatedPinout"] = out


# Example -> find_next_number("AI 22", 0) -> 22
# Example -> find_next_number("AI 2455", 2) -> 242424245
def find_next_number(s: str, start: int) -> int | None:
    num = ""
    for i in range(start, len(s)):
        if s[i].isdigit():
            num += s[i]
        elif len(num) > 0:
            break
    if len(num) == 0:
        return None
    return int(num)


def maybe_update_lines_at_index(items: list[int], port: int, mbe_line_max: int = 1):
    # if the index is greater than the length of the list extend it with zero
    if port >= len(items):
        items.extend([0] * (port - len(items) + 1))
    if mbe_line_max > items[port]:
        items[port] = mbe_line_max


NO_UPPER = -1


class PinoutParser(HTMLParser):
    # ANALOG OUT
    ao_max: int

    # ANALOG IN
    ai_max: int

    # THERMOCOUPLE
    tc_max: int

    # DIGITAL I/O
    dio_ports: list[int]

    # DIGITAL IN
    di_ports: list[int]

    # DIGITAL OUT
    do_ports: list[int]

    curr_tag: str = ""

    def __init__(self):
        super().__init__()
        self.ao_max = NO_UPPER
        self.ai_max = NO_UPPER
        self.tc_max = NO_UPPER
        self.dio_ports = []
        self.di_ports = []
        self.do_ports = []

    def handle_starttag(self, tag, attrs, **kwargs):
        if tag == "text":
            self.curr_tag = "text"

    def handle_endtag(self, tag):
        if tag == "text":
            self.curr_tag = ""

    def handle_data(self, data):
        if self.curr_tag == "text":
            if "AI" in data:
                n = find_next_number(data, 0)
                if n is None:
                    return
                self.ai_max = max(self.ai_max, n)
            elif "AO" in data:
                n = find_next_number(data, 0)
                if n is None:
                    return
                self.ao_max = max(self.ao_max, n)
            elif "DIO" in data:
                # check if the next letter is a letter in the alphabet
                # if it is, then it's a port
                dio_idx = data.find("DIO")
                try:
                    char = data[dio_idx + 3]
                except IndexError:
                    return
                is_alphabet = char.isalpha()
                search_idex = dio_idx + 3
                port = 0
                if is_alphabet:
                    port = ord(char) - ord("A")
                    search_idex += 1
                    maybe_update_lines_at_index(self.dio_ports, port, 0)
                n = find_next_number(data, search_idex)
                if n is None:
                    return
                maybe_update_lines_at_index(self.dio_ports, port, n)
            elif "DI" in data:
                n = find_next_number(data, 0)
                if n is None:
                    return
                maybe_update_lines_at_index(self.di_ports, 0, n)
            elif "DO" in data:
                n = find_next_number(data, 0)
                if n is None:
                    return
                maybe_update_lines_at_index(self.do_ports, 0, n)
            elif "TC" in data:
                n = find_next_number(data, 0)
                if n is None:
                    return
                self.tc_max = max(self.tc_max, n)
            else:
                # find all occurrences of "P" in data
                # find the number after each "P"
                for i in range(len(data) - 1):
                    if data[i] == "P" and data[i + 1].isdigit():
                        port = find_next_number(data, i + 1)
                        if port is None:
                            return
                        maybe_update_lines_at_index(self.dio_ports, port, 0)
                        if data[i + 1 + len(str(port))] == ".":
                            mbe_line_max = find_next_number(
                                data,
                                i + 1 + len(str(port)) + 1,
                            )
                            if mbe_line_max is None:
                                return
                            maybe_update_lines_at_index(
                                self.dio_ports, port, mbe_line_max
                            )


headers = {
    "Client_id": "1fcca773908c4e6da0500a60ea393e83",
    "Client_secret": "9EC7AA4494614C25AE57f022Bc6f7Bac",
    "Referer": "https://www.ni.com/",
}


def parse_pinout_info(url: str):
    response = requests.get(url, headers=headers)
    if response.status_code != 200:
        warnings.warn(f"Failed to get pinout info for {url}")
        if response.status_code == 503:
            time.sleep(1)
            return parse_pinout_info(url)
        return
    parser = PinoutParser()
    parser.feed(response.text)
    out = dict()
    out["analogOut"] = {"portCount": parser.ao_max + 1}
    out["analogIn"] = {"portCount": parser.ai_max + 1}
    out["thermocouple"] = {"portCount": parser.tc_max + 1}
    out["digitalInputOutput"] = {
        "portCount": len(parser.dio_ports),
        "lineCounts": [x + 1 for x in parser.dio_ports],
    }
    out["digitalInput"] = {
        "portCount": len(parser.di_ports),
        "lineCounts": [x + 1 for x in parser.di_ports],
    }
    out["digitalOutput"] = {
        "portCount": len(parser.do_ports),
        "lineCounts": [x + 1 for x in parser.do_ports],
    }

    if len(out) == 0:
        return None

    return out


def process_pinout(i, product):
    if "pinouts" not in product:
        return
    pinouts = product["pinouts"]
    for pout in pinouts:
        if "img" not in pout:
            continue
        url = pout["img"]
        pinout_info = parse_pinout_info(url)
        if pinout_info is None:
            continue
        pout["info"] = pinout_info
        estimate_pinout_accuracy(i, product)
    return product


in_products = json.load(open("out/products.json", "r"))

for product in in_products:
    if "pinouts" in product:
        for pinout in product["pinouts"]:
            if "info" in pinout:
                del pinout["info"]

process_products(in_products, process_pinout, 10)
json.dump(in_products, open("out/products.json", "w"))

INPUT_ONLY_CHANNELS = "NumberOfInputOnlyChannels"
OUTPUT_ONLY_CHANNELS = "NumberOfOutputOnlyChannels"
INPUT_OUTPUT_CHANNELS = "NumberOfBidirectionalChannels"

# 1 - Reference Oriented

# Concepts

# Python Client

# - Guide on processing historical data

# Console

# 2 - Persona Oriented

# Operations

# - What you need know conceptually operate>

# Analysis

# Administration

# Development
