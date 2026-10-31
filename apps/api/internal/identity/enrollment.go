package identity

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

type EnrollmentNumberGenerator interface {
	Generate() (string, error)
}

type DefaultEnrollmentGenerator struct {
	clock func() time.Time
}

func NewEnrollmentGenerator(clock func() time.Time) *DefaultEnrollmentGenerator {
	if clock == nil {
		clock = time.Now
	}
	return &DefaultEnrollmentGenerator{clock: clock}
}

var fortalezaLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/Fortaleza")
	if err != nil {
		return time.FixedZone("America/Fortaleza", -3*60*60)
	}
	return loc
}()

func (g *DefaultEnrollmentGenerator) Generate() (string, error) {
	year := g.clock().In(fortalezaLocation).Year()

	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}

	return fmt.Sprintf("%04d%06d", year, n.Int64()), nil
}
