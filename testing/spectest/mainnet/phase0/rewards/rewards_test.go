package rewards

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/phase0/rewards"
)

func TestMainnet_Phase0_Rewards(t *testing.T) {
	rewards.RunPrecomputeRewardsAndPenaltiesTests(t, "mainnet")
}
