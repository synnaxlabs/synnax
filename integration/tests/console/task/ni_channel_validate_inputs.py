#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import platform
import random
import synnax as sy

from console.case import ConsoleCase


class Ni_Channel_Validate_Inputs(ConsoleCase):
    """
    Test the input selection for each channel type. Not running the tasks here. 
    Only veryifying that each input type (dropdown/int/float) can be 
    appropriately selected.
    """

    # TODO:
    # - Validate bad inputs for alll types
    # - Validate custom scale inputs
    # - And basically all tasks
    # - Step through each pre-configured channel and get info
    #
    # - "name" will be used to rename channels AFTER we have 
    #   created a task. (not for this test)

    def setup(self) -> None:
        if platform.system() != "Windows":
           #self.auto_pass(msg="Requires DAQmx drivers")
           print('autopass goes here')
        super().setup()

    def run(self) -> None:
        """
        Test Opening and closing pages
        """
        console = self.console
        client = self.client

        #
        # Remove the following when ready to
        # Talk to NI MAX sim devices
        ###########################
        rack_name = f"TestRack_{random.randint(100, 999)}"
        device_name = "E103"
        self._log_message("Creating NI Analog Read Task")
        console.task.new()

        # Check simple functionality
        console.task.set_parameters(
            task_name="Test_task",
            sample_rate=100,
            stream_rate=20,
            data_saving=True,
            auto_start=False,
        )

        self.create_test_rack(rack_name, device_name)
        #self.validate_voltage_inputs(device_name)
        #self.validate_accel_inputs(device_name)
        #self.validate_bridge_inputs(device_name)
        #self.validate_current_inputs(device_name)
        self.validate_force_bridge_table_inputs(device_name)

        sy.sleep(20)

    def create_test_rack(self, rack_name: str, device_name: str) -> None:
        rack = self.client.hardware.racks.create(name=rack_name)
        self.client.hardware.devices.create(
            [
                sy.Device(
                    key="130227d9-02aa-47e4-b370-0d590add1bc1",
                    rack=rack.key,
                    name=device_name,
                    make="NI",
                    model="NI 9229",
                    location=device_name,
                    identifier=f"{device_name}Mod1",
                )
            ]
        )
        sy.sleep(1)
            
    def validate_voltage_inputs(self, device_name: str) -> None:
        """ Validate voltage inputs """
        console = self.console
        type = "Voltage"

        console.task.add_channel(
            name = "v0", 
            type = type,
            device = device_name,
            terminal_config = "Default",
        )
        console.task.add_channel(
            name = "v1",
            type = type,
            device=device_name,
            terminal_config = "Differential",
            min_val = -0.1,
            max_val = 6.5,
        )
        console.task.add_channel(
            name = "v2",
            type = type,
            device=device_name,
            terminal_config = "Pseudo-Differential",
            min_val = -10,
            max_val = 10,
        )
        console.task.add_channel(
            name = "v3",
            type = type,
            device=device_name,
            terminal_config = "Referenced Single Ended",
        )
        console.task.add_channel(
            name = "v4",
            type = type,
            device=device_name,
            terminal_config = "Non-Referenced Single Ended",
        )
    
    def validate_accel_inputs(self, device_name: str) -> None:
        """ Validate accel inputs """
        console = self.console
        type = "Accelerometer"

        console.task.add_channel(
            name="Accel_1",
            type=type,
            device=device_name,
        )
        console.task.add_channel(
            name="Accel_2",
            type=type,
            device=device_name,
            sensitivity=0.25,
            units="mV/g",
            excitation_source="Internal",
            current_excitation_value=0.1
        )
        console.task.add_channel(
            name="Accel_3",
            type=type,
            device=device_name,
            units="V/g",
            excitation_source="External",
        )
        console.task.add_channel(
            name="Accel_4",
            type=type,
            device=device_name,
            excitation_source="None",
        )
        
    def validate_bridge_inputs(self, device_name: str) -> None:
        """ Validate Bridge inputs """
        console = self.console
        type = "Bridge"

        console.task.add_channel(
            name="Bridge_1",
            type=type,
            device=device_name,
        )
        console.task.add_channel(
            name="Bridge_2",
            type=type,
            device=device_name,
            units="mV/V",
            configuration="Full Bridge",
            resistance=0.1,
            excitation_source="Internal",
            excitation_value=0.2,
        )
        console.task.add_channel(
            name="Bridge_3",
            type=type,
            device=device_name,
            units="V/V",
            configuration="Half Bridge",
            excitation_source="External",
        ) 
        console.task.add_channel(
            name="Bridge_4",
            type=type,
            device=device_name,
            configuration="Quarter Bridge",
            excitation_source="None",
        )

    def validate_current_inputs(self, device_name: str) -> None:
        """ Validate Bridge inputs """
        console = self.console
        type = "Current"

        console.task.add_channel(
            name="Current_1",
            type=type,
            device=device_name,
        )
        console.task.add_channel(
            name="Current_2",
            type=type,
            device=device_name,
            shunt_resistor="Default",
            resistance=0.1,
        )
        console.task.add_channel(
            name="Current_3",
            type=type,
            device=device_name,
            shunt_resistor = "Internal"
        ) 
        console.task.add_channel(
            name="Bridge_4",
            type=type,
            device=device_name,
            shunt_resistor= "External",
        )

    def validate_force_bridge_table_inputs(self, device_name: str) -> None:
        """ Validate Force Bridge Table inputs """
        console = self.console
        type = "Force Bridge Table"

        console.task.add_channel(
            name="ForceBridge_1",
            type=type,
            device=device_name,
        )
        console.task.add_channel(
            name="ForceBridge_2",
            type=type,
            device=device_name,
            force_units="Newtons",
            bridge_configuration="Full Bridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Newtons",
            electrical_units="mV/V",
        )
        console.task.add_channel(
            name="ForceBridge_3",
            type=type,
            device=device_name,
            force_units="Pounds",
            bridge_configuration="Half Bridge",
            excitation_source="External",
            physical_units="Pounds",
            electrical_units="V/V",
        ),
        console.task.add_channel(
            name="ForceBridge_3",
            type=type,
            device=device_name,
            force_units="Kilograms",
            physical_units="Kilograms",
        )
