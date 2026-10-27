#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fleet spend, fetched and written once every INTERVAL. Cost Explorer gives the
account, region, service, and instance type totals, usage only and with credits
(_net), plus the month forecast. List prices and CloudWatch hours give each runner's
estimate. Sums are calculated channels. Data Exports to S3 is the path to
per-instance actuals."""

import datetime as dt
import json
import math
import re
from collections.abc import Callable
from concurrent.futures import Future, ThreadPoolExecutor
from typing import Any

import synnax as sy

from aws import AWS, REGIONS
from common import channel_name, instance_for, instance_prefix, latest
from log import Log

QUERY_TIME = "cost_query_time"
MONTH_REMAINING_FORECAST = "cost_month_remaining_forecast"
HOURS_REMAINING = "month_hours_remaining"
# Per-runner suffixes.
RATE, EBS, EST_1D, EST_MTD = (
    "cost_per_hour",
    "ebs_cost_per_hour",
    "cost_1d_est",
    "cost_mtd_est",
)
PER_RUNNER = (RATE, EBS, EST_1D, EST_MTD)
HOURS_PER_MONTH = 730
# EC2 reports one CPUUtilization sample a minute while running.
SAMPLES_PER_HOUR = 60
INTERVAL = 6 * sy.TimeSpan.HOUR
DAYS = 365
GLOBAL = "global"
SERVICES = {
    "Amazon Elastic Compute Cloud - Compute": "ec2",
    "EC2 - Other": "ec2_other",
    "Amazon Simple Storage Service": "s3",
    "AWS Systems Manager": "ssm",
    "AmazonCloudWatch": "cloudwatch",
}
OTHER = "other"
WINDOWS = ("1d", "7d", "mtd", "30d", "90d", "lm", "ytd", "365d")
COST, USAGE = "UnblendedCost", "UsageQuantity"
REGION = {"Type": "DIMENSION", "Key": "REGION"}
SERVICE = {"Type": "DIMENSION", "Key": "SERVICE"}
INSTANCE_TYPE = {"Type": "DIMENSION", "Key": "INSTANCE_TYPE"}
RECORD_TYPE = {"Type": "DIMENSION", "Key": "RECORD_TYPE"}
USAGE_RECORD = "Usage"
# A type channel is cost_<family>_<size>_<window>, the family a letter then a digit.
TYPE_CHANNEL = re.compile(r"^cost_([a-z]\d[a-z\d]*_[a-z\d]+)_365d$")
OUR_REGIONS = {"Dimensions": {"Key": "REGION", "Values": list(REGIONS)}}
USAGE_ONLY: Any = {"Dimensions": {"Key": "RECORD_TYPE", "Values": [USAGE_RECORD]}}

Group = tuple[str, ...]
# Maps a row's keys to its group, None drops the row.
Key = Callable[[list[str]], Group | None]


def name(*parts: str) -> str:
    return "_".join(("cost", *parts))


def groups() -> list[Group]:
    """Every channel prefix, without the window."""
    regions = [*REGIONS.values(), GLOBAL]
    services = [*SERVICES.values(), OTHER]
    return [
        (),
        *((r,) for r in regions),
        *((r, s) for r in REGIONS.values() for s in services),
    ]


def hours(region: str, window: str) -> str:
    return f"hours_{region}_{window}"


def type_key(instance_type: str) -> str:
    return re.sub(r"[^a-z0-9]+", "_", instance_type.lower())


def month_end(day: dt.date) -> dt.datetime:
    """00:00 UTC on the 1st of the next month."""
    first = dt.datetime(day.year, day.month, 1, tzinfo=dt.UTC)
    return (first + dt.timedelta(days=32)).replace(day=1)


class Prices:
    """On-demand list prices, one GetProducts per distinct lookup, cached."""

    def __init__(self, aws: AWS) -> None:
        self.pricing = aws.pricing
        self.cache: dict[tuple, float] = {}

    def usd(self, **filters: str) -> float:
        key = tuple(sorted(filters.items()))
        if key not in self.cache:
            page = self.pricing.get_products(
                ServiceCode="AmazonEC2",
                Filters=[
                    {"Type": "TERM_MATCH", "Field": k, "Value": v}
                    for k, v in filters.items()
                ],
                MaxResults=2,
            )
            if len(page["PriceList"]) != 1:
                raise LookupError(f"{len(page['PriceList'])} products for {filters}")
            terms = json.loads(page["PriceList"][0])["terms"]["OnDemand"]
            dims = next(iter(terms.values()))["priceDimensions"]
            self.cache[key] = float(next(iter(dims.values()))["pricePerUnit"]["USD"])
        return self.cache[key]

    def instance(self, instance_type: str, region: str, windows: bool) -> float:
        """USD per hour."""
        return self.usd(
            instanceType=instance_type,
            regionCode=region,
            operatingSystem="Windows" if windows else "Linux",
            tenancy="Shared",
            capacitystatus="Used",
            preInstalledSw="NA",
            licenseModel="No License required",
        )

    def volume(self, region: str) -> float:
        """USD per GiB-hour."""
        return (
            self.usd(productFamily="Storage", volumeApiName="gp3", regionCode=region)
            / HOURS_PER_MONTH
        )


def amounts(rows: list[dict], key: Key, metric: str) -> dict[Group, dict[str, float]]:
    """Daily amounts of one metric per group, groups that map to the same key
    summed."""
    out: dict[Group, dict[str, float]] = {}
    for row in rows:
        date = row["TimePeriod"]["Start"]
        for g in row.get("Groups", []):
            if (group := key(g["Keys"])) is None:
                continue
            days = out.setdefault(group, {})
            days[date] = days.get(date, 0.0) + float(g["Metrics"][metric]["Amount"])
    return out


def fold(
    queries: list[tuple[list[dict], Key]], metric: str
) -> dict[Group, dict[str, float]]:
    out: dict[Group, dict[str, float]] = {}
    for rows, key in queries:
        out |= amounts(rows, key, metric)
    return out


def windows(days: dict[str, float], today: dt.date) -> dict[str, float]:
    def back(n: int) -> float:
        return sum(
            days.get((today - dt.timedelta(days=i)).isoformat(), 0.0)
            for i in range(1, n + 1)
        )

    def within(prefix: str) -> float:
        return sum(v for d, v in days.items() if d.startswith(prefix))

    last_month = today.replace(day=1) - dt.timedelta(days=1)
    return {
        "1d": back(1),
        "7d": back(7),
        "mtd": within(today.strftime("%Y-%m")),
        "30d": back(30),
        "90d": back(90),
        "lm": within(last_month.strftime("%Y-%m")),
        "ytd": within(today.strftime("%Y-")),
        "365d": back(365),
    }


def spend_names(net: bool) -> list[str]:
    """Account, region, and service channels, plus the hours when gross."""
    tail = ("net",) if net else ()
    names = [name(*g, w, *tail) for g in groups() for w in WINDOWS]
    if not net:
        names += [hours(r, w) for r in REGIONS.values() for w in WINDOWS]
    return names


def type_names(types: set[str]) -> list[str]:
    return [
        name(t, w, *net)
        for t in sorted(types)
        for w in WINDOWS
        for net in ((), ("net",))
    ]


def runner_names(known: list[str]) -> list[str]:
    return [channel_name(s, r) for r in known for s in PER_RUNNER]


class Cost:
    def __init__(self, log: Log, aws: AWS) -> None:
        self.log = log
        self.aws = aws
        self.prices = Prices(aws)
        # Runners with an instance behind them, as of the last step.
        self.known: list[str] = []
        # Instance types seen, from the Core at startup and from every fetch.
        self.types: set[str] = set()
        self.attempted: sy.TimeStamp | None = None
        # The fetch runs off the controller cycle; step() takes its result.
        self.pool = ThreadPoolExecutor(1)
        self.pending: Future[dict[str, float]] | None = None
        # Each priced runner's instance, list price, and volume share, from the last
        # fetch, so rates follow starts and stops between fetches.
        self.priced: dict[str, tuple[str, float, float]] = {}
        self.rates: dict[str, float] = {}

    def channels(self) -> dict[str, sy.DataType]:
        """Every data channel, all on the QUERY_TIME index."""
        names = [
            *spend_names(False),
            *spend_names(True),
            *type_names(self.types),
            *runner_names(self.known),
            MONTH_REMAINING_FORECAST,
            HOURS_REMAINING,
        ]
        return dict.fromkeys(names, sy.DataType.FLOAT32)

    def calculated(self, found: dict[str, dict]) -> dict[str, str]:
        """Name to expression. Regions sum their runners, the fleet sums regions."""
        by_region: dict[str, list[str]] = {r: [] for r in REGIONS.values()}
        for r in self.known:
            if (iid := instance_for(r, found)) is not None:
                by_region[REGIONS[found[iid]["region"]]].append(r)

        def total(terms: list[str]) -> str:
            return "return " + (" + ".join(terms) if terms else "0")

        out: dict[str, str] = {}
        for s in (RATE, EST_1D, EST_MTD):
            tail = s.removeprefix("cost_")
            for region, runners in by_region.items():
                out[name(region, tail)] = total([channel_name(s, r) for r in runners])
            out[name(tail)] = total([name(region, tail) for region in by_region])
        for w in ("1d", "mtd"):
            out[name(w, "est_error")] = f"return {name(w, 'est')} - {name(w)}"
        out[name("month", "forecast")] = (
            f"return {name('mtd')} + {MONTH_REMAINING_FORECAST}"
        )
        out[name("month", "est")] = (
            f"return {name('mtd')} + {name('per_hour')} * {HOURS_REMAINING}"
        )
        out[name("month", "forecast_error")] = (
            f"return {name('month', 'est')} - {name('month', 'forecast')}"
        )
        return out

    def create_state(self, client: sy.Synnax, known: set[str]) -> None:
        """The types on the Core and the last fetch time, so the cadence holds across
        a restart."""
        self.known = sorted(r for r in known if instance_prefix(r) is not None)
        try:
            for ch in client.channels.retrieve(["^cost_.*_365d$"]):
                if m := TYPE_CHANNEL.match(ch.name):
                    self.types.add(m.group(1))
            found = latest(client, [QUERY_TIME])
        except Exception as e:
            self.log.error("cost state", e)
            return
        if QUERY_TIME in found:
            self.attempted = sy.TimeStamp(int(found[QUERY_TIME]))

    def due(self, now: sy.TimeStamp) -> bool:
        return self.attempted is None or now.span(self.attempted) >= INTERVAL

    def moment(self) -> dt.datetime:
        return dt.datetime.now(dt.UTC)

    def forecast(self, today: dt.date) -> float:
        """Usage from today to the end of the month, as AWS predicts it."""
        end = month_end(today).date()
        page = self.aws.ce.get_cost_forecast(
            TimePeriod={"Start": today.isoformat(), "End": end.isoformat()},
            Metric="UNBLENDED_COST",
            Granularity="MONTHLY",
            Filter=USAGE_ONLY,
        )
        return float(page["Total"]["Amount"])

    def hours_ran(
        self, instances: dict[str, dict], start: dt.datetime, end: dt.datetime
    ) -> dict[str, float]:
        """Hours each instance ran between start and end, from its CPU sample count."""
        ran: dict[str, float] = dict.fromkeys(instances, 0.0)
        if start >= end:
            return ran
        for region, cw in self.aws.cloudwatch.items():
            ids = [i for i, info in instances.items() if info["region"] == region]
            if not ids:
                continue
            queries = [
                {
                    "Id": f"q{n}",
                    "MetricStat": {
                        "Metric": {
                            "Namespace": "AWS/EC2",
                            "MetricName": "CPUUtilization",
                            "Dimensions": [{"Name": "InstanceId", "Value": iid}],
                        },
                        "Period": 3600,
                        "Stat": "SampleCount",
                    },
                }
                for n, iid in enumerate(ids)
            ]
            params: dict = {
                "MetricDataQueries": queries,
                "StartTime": start,
                "EndTime": end,
            }
            while True:
                page = cw.get_metric_data(**params)
                for result in page["MetricDataResults"]:
                    iid = ids[int(result["Id"][1:])]
                    ran[iid] += sum(result["Values"]) / SAMPLES_PER_HOUR
                if not (token := page.get("NextToken")):
                    break
                params["NextToken"] = token
        return ran

    def estimate(self, moment: dt.datetime, found: dict[str, dict]) -> dict[str, float]:
        """Rates and the estimated spend of each runner, from list prices."""
        today = moment.date()
        midnight = dt.datetime(today.year, today.month, today.day, tzinfo=dt.UTC)
        first = midnight.replace(day=1)
        mapped = {
            r: iid for r in self.known if (iid := instance_for(r, found)) is not None
        }
        instances = {iid: found[iid] for iid in mapped.values()}
        volumes = self.aws.volumes()
        yesterday = self.hours_ran(instances, midnight - dt.timedelta(days=1), midnight)
        month = self.hours_ran(instances, first, midnight)
        # Runners without an instance cost nothing.
        values = dict.fromkeys(runner_names(self.known), 0.0)
        for r, iid in mapped.items():
            info = found[iid]
            try:
                ebs = volumes.get(iid, 0) * self.prices.volume(info["region"])
                price = self.prices.instance(
                    info["type"], info["region"], info["windows"]
                )
            except Exception as e:
                self.log.error(f"price {info['type']} {info['region']}", e)
                for s in PER_RUNNER:
                    values[channel_name(s, r)] = math.nan
                continue
            self.priced[r] = (iid, price, ebs)
            values[channel_name(RATE, r)] = ebs + (price if info["running"] else 0.0)
            values[channel_name(EBS, r)] = ebs
            values[channel_name(EST_1D, r)] = price * yesterday[iid] + ebs * 24
            values[channel_name(EST_MTD, r)] = price * month[iid] + ebs * 24 * (
                today.day - 1
            )
        return values

    def query(
        self, today: dt.date, group_by: list[dict], filter: dict | None
    ) -> list[dict]:
        params: dict = {
            "TimePeriod": {
                "Start": (today - dt.timedelta(days=DAYS)).isoformat(),
                "End": today.isoformat(),
            },
            "Granularity": "DAILY",
            "Metrics": [COST, USAGE],
        }
        if group_by:
            params["GroupBy"] = group_by
        if filter:
            params["Filter"] = filter
        rows: list[Any] = []
        while True:
            page = self.aws.ce.get_cost_and_usage(**params)
            rows += page["ResultsByTime"]
            if not (token := page.get("NextPageToken")):
                return rows
            params["NextPageToken"] = token

    def queries(
        self, today: dt.date, filter: dict | None
    ) -> list[tuple[list[dict], Key]]:
        """Regions and services under one filter. Every region is a group, so the
        regions summed are the total."""
        leaves = {"And": [filter, OUR_REGIONS]} if filter else OUR_REGIONS
        by_region = self.query(today, [REGION], filter)
        return [
            (by_region, lambda keys: ()),
            (by_region, lambda keys: (REGIONS.get(keys[0], GLOBAL),)),
            (
                self.query(today, [REGION, SERVICE], leaves),
                lambda keys: (
                    REGIONS.get(keys[0], GLOBAL),
                    SERVICES.get(keys[1], OTHER),
                ),
            ),
        ]

    def spend(self, today: dt.date, net: bool) -> dict[str, float]:
        """Account, region, and service windows, plus the hours when gross."""
        queries = self.queries(today, None if net else USAGE_ONLY)
        tail = ("net",) if net else ()
        cost = fold(queries, COST)
        values = {
            name(*g, w, *tail): v
            for g in groups()
            for w, v in windows(cost.get(g, {}), today).items()
        }
        if not net:
            usage = fold(queries, USAGE)
            for r in REGIONS.values():
                for w, v in windows(usage.get((r, "ec2"), {}), today).items():
                    values[hours(r, w)] = v
        return values

    def by_type(self, today: dt.date) -> dict[str, float]:
        """Windows per instance type. Grouped by record type as well, the usage rows
        alone are the gross."""
        rows = self.query(today, [INSTANCE_TYPE, RECORD_TYPE], None)
        gross = amounts(
            rows,
            lambda keys: (type_key(keys[0]),) if keys[1] == USAGE_RECORD else None,
            COST,
        )
        net = amounts(rows, lambda keys: (type_key(keys[0]),), COST)
        # Usage with no instance type groups under a placeholder key.
        types = {g[0] for g in gross if TYPE_CHANNEL.match(name(g[0], "365d"))}
        values: dict[str, float] = {}
        for t in sorted(types | self.types):
            for w, v in windows(gross.get((t,), {}), today).items():
                values[name(t, w)] = v
            for w, v in windows(net.get((t,), {}), today).items():
                values[name(t, w, "net")] = v
        return values

    def part(
        self, what: str, names: list[str], get: Callable[[], dict[str, float]]
    ) -> dict[str, float]:
        """The part's values, or NaN on its channels when it fails."""
        try:
            return get()
        except Exception as e:
            self.log.error(f"cost {what}", e)
            return dict.fromkeys(names, math.nan)

    def fetch(self, moment: dt.datetime, found: dict[str, dict]) -> dict[str, float]:
        today = moment.date()
        values = self.part(
            "spend", spend_names(False), lambda: self.spend(today, False)
        )
        values |= self.part("net", spend_names(True), lambda: self.spend(today, True))
        values |= self.part(
            "types", type_names(self.types), lambda: self.by_type(today)
        )
        values |= self.part(
            "forecast",
            [MONTH_REMAINING_FORECAST],
            lambda: {MONTH_REMAINING_FORECAST: self.forecast(today)},
        )
        values |= self.part(
            "estimate", runner_names(self.known), lambda: self.estimate(moment, found)
        )
        values[HOURS_REMAINING] = (month_end(today) - moment).total_seconds() / 3600
        return values

    def current_rates(self, found: dict[str, dict]) -> dict[str, float]:
        """Each priced runner's rate whose instance started or stopped since the
        rate was last returned."""
        changed: dict[str, float] = {}
        for r, (iid, price, ebs) in self.priced.items():
            info = found.get(iid)
            if info is None:
                continue
            rate = ebs + (price if info["running"] else 0.0)
            if self.rates.get(r) != rate:
                changed[channel_name(RATE, r)] = rate
                self.rates[r] = rate
        return changed

    def step(
        self, now: sy.TimeStamp, known: set[str], found: dict[str, dict]
    ) -> dict[str, float]:
        """The values on the cycle a fetch lands, else the rates that changed since the
        last cycle, from the last fetch's prices."""
        self.known = sorted(r for r in known if instance_prefix(r) is not None)
        if self.pending is None and self.due(now):
            self.attempted = now
            self.pending = self.pool.submit(self.fetch, self.moment(), found)
        if self.pending is None or not self.pending.done():
            return self.current_rates(found)
        try:
            values = self.pending.result()
        except Exception as e:
            self.log.error("cost fetch", e)
            return self.current_rates(found)
        finally:
            self.pending = None
        self.rates = {
            r: v
            for r in self.priced
            if (v := values.get(channel_name(RATE, r))) is not None
        }
        self.types |= {m.group(1) for n in values if (m := TYPE_CHANNEL.match(n))}
        self.log(f"cost fetched, mtd ${values[name('mtd')]:.2f}")
        return values

    def close(self) -> None:
        self.pool.shutdown(wait=False)
