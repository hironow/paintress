package domain

import (
	"fmt"
	"math"
)

// SPRTVerdict represents the outcome of a Sequential Probability Ratio Test.
type SPRTVerdict string

const (
	SPRTPass         SPRTVerdict = "PASS"
	SPRTFail         SPRTVerdict = "FAIL"
	SPRTInconclusive SPRTVerdict = "INCONCLUSIVE"
)

// SPRTConfig holds parameters for the Sequential Probability Ratio Test.
// Based on AgentAssay (arXiv:2603.02601) defaults.
type SPRTConfig struct { // nosemgrep: structure.multiple-exported-structs-go -- SPRT family; SPRTConfig co-locates with SPRTVerdict constants and DefaultSPRTConfig; splitting would fragment the AgentAssay statistical testing module [permanent]
	P0    float64 // null hypothesis success rate (regression threshold)
	P1    float64 // alternative hypothesis success rate (expected)
	Alpha float64 // type I error rate (false FAIL)
	Beta  float64 // type II error rate (false PASS)
}

// DefaultSPRTConfig returns AgentAssay's recommended defaults.
func DefaultSPRTConfig() SPRTConfig {
	return SPRTConfig{
		P0:    0.70,
		P1:    0.85,
		Alpha: 0.05,
		Beta:  0.05,
	}
}

// SPRTState holds the intermediate state of an SPRT computation.
type SPRTState struct {
	Successes  int
	Failures   int
	LambdaN    float64 // log-likelihood ratio
	UpperBound float64 // A = log((1-beta)/alpha)
	LowerBound float64 // B = log(beta/(1-alpha))
}

// ParseSPRTConfig validates an SPRTConfig and returns the validated config or an error.
// It checks that values will not produce NaN/Inf in the log-likelihood computation.
func ParseSPRTConfig(cfg SPRTConfig) (SPRTConfig, error) {
	if cfg.P0 <= 0 || cfg.P0 >= 1 {
		return SPRTConfig{}, fmt.Errorf("P0 must be in (0,1), got %g", cfg.P0)
	}
	if cfg.P1 <= 0 || cfg.P1 >= 1 {
		return SPRTConfig{}, fmt.Errorf("P1 must be in (0,1), got %g", cfg.P1)
	}
	if cfg.P0 >= cfg.P1 {
		return SPRTConfig{}, fmt.Errorf("P0 must be less than P1 (got P0=%g, P1=%g)", cfg.P0, cfg.P1)
	}
	if cfg.Alpha <= 0 || cfg.Alpha >= 1 {
		return SPRTConfig{}, fmt.Errorf("Alpha must be in (0,1), got %g", cfg.Alpha)
	}
	if cfg.Beta <= 0 || cfg.Beta >= 1 {
		return SPRTConfig{}, fmt.Errorf("Beta must be in (0,1), got %g", cfg.Beta)
	}
	return cfg, nil
}

// ValidateSPRTConfig checks that cfg values will not produce NaN/Inf in
// the log-likelihood computation. Returns an error describing the violation.
//
// Deprecated: prefer ParseSPRTConfig which returns the validated config.
func ValidateSPRTConfig(cfg SPRTConfig) error { // nosemgrep: parse-dont-validate.validate-returns-error-only-go -- backward-compat wrapper; ParseSPRTConfig is the canonical parse function [permanent]
	_, err := ParseSPRTConfig(cfg)
	return err
}

// SPRT runs a Sequential Probability Ratio Test on expedition results.
// Each result is true (success) or false (failure).
// Returns the verdict and the final state.
func SPRT(results []bool, cfg SPRTConfig) (SPRTVerdict, SPRTState) {
	upperBound := math.Log((1 - cfg.Beta) / cfg.Alpha)
	lowerBound := math.Log(cfg.Beta / (1 - cfg.Alpha))

	logP1overP0 := math.Log(cfg.P1 / cfg.P0)
	logQ1overQ0 := math.Log((1 - cfg.P1) / (1 - cfg.P0))

	var successes, failures int
	lambdaN := 0.0

	for _, success := range results {
		if success {
			successes++
			lambdaN += logP1overP0
		} else {
			failures++
			lambdaN += logQ1overQ0
		}

		if lambdaN >= upperBound {
			return SPRTPass, SPRTState{
				Successes: successes, Failures: failures,
				LambdaN: lambdaN, UpperBound: upperBound, LowerBound: lowerBound,
			}
		}
		if lambdaN <= lowerBound {
			return SPRTFail, SPRTState{
				Successes: successes, Failures: failures,
				LambdaN: lambdaN, UpperBound: upperBound, LowerBound: lowerBound,
			}
		}
	}

	return SPRTInconclusive, SPRTState{
		Successes: successes, Failures: failures,
		LambdaN: lambdaN, UpperBound: upperBound, LowerBound: lowerBound,
	}
}
