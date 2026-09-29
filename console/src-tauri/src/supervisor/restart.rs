// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! The restart policy for the embedded Core: restart with backoff after an unexpected
//! exit, and give up when the Core crash-loops.

use std::time::Duration;

/// Configures a [`Policy`].
#[derive(Clone, Copy, Debug)]
pub struct Config {
    /// The backoff before the first restart. It grows by `scale` on each consecutive
    /// failed run.
    pub base_interval: Duration,
    /// The multiplicative growth of the backoff per consecutive failed run.
    pub scale: f64,
    /// The number of consecutive failed runs after which the policy gives up.
    pub max_retries: u32,
    /// A run at least this long is healthy: its exit resets the failure count.
    pub healthy_uptime: Duration,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            base_interval: Duration::from_secs(2),
            scale: 1.1,
            max_retries: 5,
            healthy_uptime: Duration::from_secs(60),
        }
    }
}

/// What the supervisor does after the Core exits unexpectedly.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum Decision {
    /// Start the Core again after the given backoff.
    Restart(Duration),
    /// Leave the Core stopped.
    GiveUp,
}

/// Decides whether to restart the Core after an unexpected exit.
#[derive(Debug)]
pub struct Policy {
    cfg: Config,
    failures: u32,
}

impl Policy {
    pub fn new(cfg: Config) -> Self {
        Self { cfg, failures: 0 }
    }

    /// Records an unexpected exit after a run of `uptime` and returns the decision.
    pub fn decide(&mut self, uptime: Duration) -> Decision {
        if uptime >= self.cfg.healthy_uptime {
            self.failures = 0;
        }
        if self.failures >= self.cfg.max_retries {
            return Decision::GiveUp;
        }
        let backoff = self
            .cfg
            .base_interval
            .mul_f64(self.cfg.scale.powi(self.failures as i32));
        self.failures += 1;
        Decision::Restart(backoff)
    }

    /// Clears the failure count.
    pub fn reset(&mut self) {
        self.failures = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn policy() -> Policy {
        Policy::new(Config {
            base_interval: Duration::from_secs(2),
            scale: 2.0,
            max_retries: 3,
            healthy_uptime: Duration::from_secs(60),
        })
    }

    #[test]
    fn grows_the_backoff_on_consecutive_failures() {
        let mut p = policy();
        assert_eq!(
            p.decide(Duration::ZERO),
            Decision::Restart(Duration::from_secs(2))
        );
        assert_eq!(
            p.decide(Duration::ZERO),
            Decision::Restart(Duration::from_secs(4))
        );
        assert_eq!(
            p.decide(Duration::ZERO),
            Decision::Restart(Duration::from_secs(8))
        );
    }

    #[test]
    fn gives_up_after_max_retries_consecutive_failures() {
        let mut p = policy();
        for _ in 0..3 {
            assert!(matches!(p.decide(Duration::ZERO), Decision::Restart(_)));
        }
        assert_eq!(p.decide(Duration::ZERO), Decision::GiveUp);
        assert_eq!(p.decide(Duration::ZERO), Decision::GiveUp);
    }

    #[test]
    fn resets_the_failure_count_after_a_healthy_run() {
        let mut p = policy();
        for _ in 0..3 {
            p.decide(Duration::ZERO);
        }
        assert_eq!(
            p.decide(Duration::from_secs(60)),
            Decision::Restart(Duration::from_secs(2))
        );
    }

    #[test]
    fn restarts_again_after_a_reset() {
        let mut p = policy();
        for _ in 0..4 {
            p.decide(Duration::ZERO);
        }
        p.reset();
        assert_eq!(
            p.decide(Duration::ZERO),
            Decision::Restart(Duration::from_secs(2))
        );
    }
}
