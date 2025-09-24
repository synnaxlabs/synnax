# 20 - Evaluating Software Reliability

**Feature Name**: Evaluating Software Reliability <br /> **Start Date**: 2024-07-11
<br /> **Authors**: Elham Islam <br />

# 0 - Summary

This serves to outline plans for improving stability to Synnax as well as maintaining an
ongoing prioritization and awareness of stability and reliability of the software we
develop. This expands on steps described on 2024-06-13 regarding
[Engineering Process Standardization](./0020-240613-engineering-workflow.md)

In short, the sources of instability and pain for the customer could come from each of
the following areas currently:

- Server
- Console
- Driver
- Cesium
- Client Libraries
- Deployment.

# 1 - Current State

Synnax's core features and functionality necessary for our ICP have largely been
developed, but still contain bugs and areas of instability. We currently are in a
position where we know there is still work to be done in both identifying and resolving
these issues. However, it feels as though there's only a nebulous sense of how large
that volume of work is and how long it will be until we get there. There are also not
clearly defined methods for addressing these bugs or identifying them beyond use.

We currently have been providing support to our customers remotely, receiving feedback
on bugs and issues as they come across them. However, I believe there is a layer of
insulation between the users of our product and their experience with the use of our
product. While being able to be on a zoom with them several times a week provides
meaningful feedback on the product, I think it is not the ideal format in which we can
get quick information on issues to diagnose. Not being on site with our users has added
additional days of engineering work to simply recreate the problem users are having. And
our current plans of being on-site largely focus on the initial deployment.

# 2 - Proposed Changes

## 2.0 - Maintenance & Software Stability vs. Feature Development

It's important before pursuing new feature development, we evaluate our set of
outstanding maintenance and stability work. The idea is to handle the higher priority
work before moving on to work that expands functionality and thereby expands the space
in which faults can arise and outpace our stability efforts.

## 2.1 - Qualification of Bugs and Stability Improvements

In order to better qualify the readiness of our software, it's important we have an
indication of the importance of any particular issue. I propose we use the following
mapping to the priority levels in linear and stay very disciplined in qualifying the
urgency of tasks in the following manner:

Urgent (1): This bug or lack of feature will stop the user from making progress on their
hardware development and testing. Could cause significant lost of time and resources for
the customer. Examples are critical bugs which can cause complete failure of a system or
subsystem of the product.

High (2): This bug or lack of feature significantly reduces quality of a user's
experience and are important to get done before any new feature development is
completed. These are bugs which significantly slow down work or could lead to repeated
work.

Medium(3): This bug or lack of feature makes a noticeable impact to the user's
experience and should be targeted to be addressed in the next release. Alternatively,
these are also bugs which have a decent workaround for them but must be addressed
eventually.

Low(4): These bugs have low impact on the user and could be put off to future releases
if necessary.

## 2.2 - Software Ownership

It is important we create clear segments of responsibility across the codebase. Having a
single owner allows for a clear path for delegation. Further, it is that engineer's
responsibility to have a mechanism or process in place to evaluate stability and
reliability of their software.

In general, it is important that everyone is responsible for their own software. That
is, if they introduce a bug or instability, it is their responsibility to address the
problem. This is ultimately is the most productive way to address issues as the RE will
have the most context for solving the problem.

# 2.3- Metrics from Active Product Use

One setback in operating in the aerospace industry and only hosting Synnax on prem is
the inability to receive logs from crashes, errors of our software. Not having a single
automated source for this information will become increasingly problematic, particularly
when customers begin to outnumber of available engineers. Further, these limits are
ability for early issue detection as issues that arise are only brought up at the
customer's discretion. While this already creates a delay between the failure event and
our ability to diagnose it, it also means we do not have information on the customer's
interpretation of such bugs. I believe it would be meaningful information to know what
bugs caused the customer to notify us immediately and which ones they were not as
concerned about.

Further, while we can expose a lot of bugs, there are a lot of patterns of use a
customer may go through for their specific application that cause bugs via unique
interactions between components; I believe it is infeasible to think that we can exhaust
these possible patterns of use through internal testing and QA. While well performed QA
can lead to a significant gains in software reliability, it is naive to solely rely on
that due to the breadth of subsystems and arrangements our software can exist in.

The exact solution to this is not clear but a couple actions that could be taken:

1. Whenever one runs into a bug, we immediately log it in as a linear issue. This has
   been essentially the current method we have been operating under but is inconsistent
   in rigor and logging all issues we see in a consistent manner.

2. Integrating monitor tooling to the software. At the very least, we can have these
   tools active in internal usage of product but also if we do have some teams like MASA
   who could potentially leave this tools active, it creates a channel of error
   communication for us. The ability to have access to a tool that immediately notifies
   us of faults would be very valuable in improving our workflow for bug handling and
   stability.

3. Finding a customer locally to be able to get more on-site support and immediate
   feedback on. This may mean finding a customer that is not exactly in our ICP, but
   mirrors the usage of Synnax in a way that still provides us meaningful information
   about the product. I believe this could be worth pursuing.

# 2.4 Setting Engineering Targets

We ultimately should be defining things similar to service-level agreements format as
internal baselines for expected quality of our product. A current issue we have is being
in a limbo of deciding when our product is polished/stable enough to go out and focus
securing more pilots/contracts. The result is being cautious in going all out in sales
in fear of delivering a premature product. By defining service level agreements for a
particular type of customer, we can have targets that allow us to say that, once we
reach a certain level of engineering work and can verify certain metrics/functionality
on the product, we can then pursue sales with that customer without concerns of
delivering a product.

Example:

- the sorts of channels we support
- time between subsequent errors
- volumes of data/writers/readers before degradation
- response times

These targets are really important and should be associated with the sort of people we
are selling to and our assumptions of what we believe to be the baseline product we need
in order to improve their experience and solve their problems to the extent that would
satisfy them (qualifying satisfaction in converted pilots for example). We need to be
careful about spending too long working on the product without getting meaningful
feedback from the users. These targets should also always include a date to reevaluate
and come back to so we do not lose track of these targets or reconsider whether they are
the correct ones.

# 2.5 Intentional Fault Injection

On a system basis, having a more intentional and comprehensive method of intentionally
causing faults. At the very least, it's important to make of all possible failure cases
a system/subsystem can be in, what causes them and whether they have a mechanism for
testing (unit testing, integration). It is ultimately the RE's job to ensure this is
done for their respective software.

# 3 Outstanding TODOS:

1. Identify REs for the major sections of the codebase
   1. Evaluate any existing holes that need to be addressed which could be:
      1. Making sure unit tests/integration tests are comprehensive if they are the best
         way to ensure stability
      2. Determining if there are unknowns in the reliability of parts of the code, how
         we can define a method for identifying those unknowns
      3. Determining a mechanism for logging errors and communicating them back to us
2. Constructing and evaluating our current list of issues and bugs and what we are
   aiming to guarantee before we make a larger push in sales.
3. Long term
   1. reliability evaluation for each of the components. Creating an internal process to
      go through a real use of our product
