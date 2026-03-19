package domain

import "math"

// SPRTVerdict represents the outcome of a Sequential Probability Ratio Test.
type SPRTVerdict string

const (
	SPRTPass         SPRTVerdict = "PASS"
	SPRTFail         SPRTVerdict = "FAIL"
	SPRTInconclusive SPRTVerdict = "INCONCLUSIVE"
)

// SPRTConfig holds parameters for the Sequential Probability Ratio Test.
// Based on AgentAssay (arXiv:2603.02601) defaults.
type SPRTConfig struct {
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
