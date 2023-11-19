# 3 - Telemetry Assembler

**Feature Name**: Telemetry Assembler <br />
**Status**: Draft <br />
**Start Date**: 2023-11-18 <br />
**Authors**: Emiliano Bonilla <br />

# 0 - Summary

It's rarely the case that raw sensor or actuator output values are directly useful to
an engineer. Typically, these values need to be converted into a more useful form
through a series of mathematical transformations. Our users most commonly request this
by saying, "Can I set up a calculation to perform on this value?". In this RFC we examine
the various ways our users are interested in transforming their data before converting
it to a visualization or analytical result, and propose a strategy for rolling out this
feature in a sustainable, progressive manner.

# 1 - Problem Statement

The current Synnax console is limited in its ability to transform incoming channel 
values before displaying them. For the most part, users are limited to viewing raw 
channel data. While this is sufficient for cases where incoming channel values are already appropriately transformed, it fails in situations where these quantities are different 
than what the user is interested in, or when the user wants to combine/merge the values of multiple channels.

For example, it's typical to apply a linear calibration equation to a pressure transducer to 
convert its raw voltage output to a pressure value. In the current Synnax console, this value 
would need to be scaled by the data acquisition device *before* being sent to the Synnax server.

Other common use cases involve combining multiple channels into a single value. Perhaps
the quintissential example is sensor voting, where the closest two of three sensors are
averaged to produce a more accurate value. 

These types of operations are so frequently used that it's practically a requirement
for any data visualization tool to support them.

# 2 - User Research Summary

As always, User research is the dirving force behind the design of this feature set. All
of our users research findings can be found [here](https://drive.google.com/drive/u/0/folders/13Vc-G5CNzCwhxx9vNsHJLECK9Mrqz0if).
As this is a public document, we've anonymized and summarized our findings below.

## 2.0 - Transformations with different levels of complexity require different tools

When doing complex analysis like a signal transformation, our users reach for advanced,
often programming-based tools like MATLAB or Python. When the complexity drops, however,
they often reach for simpler, graphical tools like Excel.

It's important to note that there is need for both of these sides of the spectrum. Making
a user interface to perform every variation of a complex analysis worklfow is unsustainable,
and would result in a bloated, confusing interfaces. On the other hand, our users don't
want to write code to get the average of two numbers.

## 2.1 - Transformations extend beyond purely numeric quantities

While the most common transformations are mathematical, our users are interested in 
seeing results displayed in a variety of ways, most notably color. For example, on our
plumbing and instrumentation diagrams, users want to see the color of a valve change
based on its state. Or they'd like colorful indicators on temperature values to show
when they're in a safe range.

## 2.2 - Transformations involve multiple channels
