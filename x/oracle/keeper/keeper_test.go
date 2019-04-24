package keeper

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swishlabsco/cosmos-ethereum-bridge/x/oracle/types"
)

func TestCreateGetProphecy(t *testing.T) {
	ctx, _, keeper, validatorAddresses, _ := CreateTestKeepers(t, false, 0.7, nil, []int64{10})
	testProphecy := types.CreateTestProphecy(validatorAddresses[0])

	//Test normal Creation
	err := keeper.CreateProphecy(ctx, testProphecy)
	require.NoError(t, err)

	//Test bad Creation
	badProphecy := types.CreateTestProphecy(validatorAddresses[0])
	badProphecy.MinimumPower = -1
	err = keeper.CreateProphecy(ctx, badProphecy)

	badProphecy2 := types.CreateTestProphecy(validatorAddresses[0])
	badProphecy2.ID = ""
	err = keeper.CreateProphecy(ctx, badProphecy2)
	require.Error(t, err)

	badProphecy3 := types.CreateTestProphecy(validatorAddresses[0])
	badProphecy3.Claims = []types.Claim{}
	err = keeper.CreateProphecy(ctx, badProphecy3)
	require.Error(t, err)

	//Test retrieval
	prophecy, err := keeper.GetProphecy(ctx, testProphecy.ID)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(testProphecy, prophecy))
}

func TestBadConsensusForOracle(t *testing.T) {
	_, _, _, _, err := CreateTestKeepers(t, false, 0, nil, []int64{10})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "Oracle consensus needed cannot be less than 0.001"))

	_, _, _, _, err = CreateTestKeepers(t, false, 1.2, nil, []int64{10})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "Oracle consensus needed cannot be greater than 1"))
}

func TestDuplicateMsgs(t *testing.T) {
	ctx, _, keeper, validatorAddresses, err := CreateTestKeepers(t, false, 0.6, nil, []int64{3, 3})
	validator1Pow3 := validatorAddresses[0]
	testClaim := types.CreateTestClaimForValidator(validator1Pow3)
	testClaimAltV1 := types.CreateAlternateTestClaimForValidator(validator1Pow3)

	//Test normal Creation
	progressUpdate, err := keeper.ProcessClaim(ctx, testClaim)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test duplicate message
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaim)
	require.Error(t, err)
	require.Equal(t, progressUpdate.Status, nil)
	require.True(t, strings.Contains(err.Error(), "Already processed message from validator for this id"))

	//Test second but non duplicate message
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimAltV1)
	require.Error(t, err)
	require.Equal(t, progressUpdate.Status, nil)
	require.True(t, strings.Contains(err.Error(), "Already processed from validator for this id"))

}

func TestSuccessfulProphecy(t *testing.T) {
	ctx, _, keeper, validatorAddresses, err := CreateTestKeepers(t, false, 0.6, nil, []int64{3, 3, 4})
	validator1Pow3 := validatorAddresses[0]
	validator2Pow3 := validatorAddresses[1]
	validator3Pow4 := validatorAddresses[2]
	testClaimV1 := types.CreateTestClaimForValidator(validator1Pow3)
	testClaimV2 := types.CreateTestClaimForValidator(validator2Pow3)
	testClaimV3 := types.CreateTestClaimForValidator(validator3Pow4)

	//Test first claim
	progressUpdate, err := keeper.ProcessClaim(ctx, testClaimV1)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test second claim completes and finalizes to success
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimV2)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.SuccessStatus)
	require.Equal(t, progressUpdate.FinalBytes, testClaimV1.ClaimBytes)

	//Test third claim not possible
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimV3)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "Prophecy already finalized"))
}

func TestSuccessfulProphecyWithDisagreement(t *testing.T) {
	ctx, _, keeper, validatorAddresses, err := CreateTestKeepers(t, false, 0.6, nil, []int64{3, 3, 4})
	validator1Pow3 := validatorAddresses[0]
	validator2Pow3 := validatorAddresses[1]
	validator3Pow4 := validatorAddresses[2]
	testClaimV1 := types.CreateTestClaimForValidator(validator1Pow3)
	testClaimAltV2 := types.CreateAlternateTestClaimForValidator(validator2Pow3)
	testClaimV3 := types.CreateTestClaimForValidator(validator3Pow4)

	//Test first claim
	progressUpdate, err := keeper.ProcessClaim(ctx, testClaimV1)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test second disagreeing claim processed fine
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimAltV2)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test third claim agrees and finalizes to success
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimV3)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.SuccessStatus)
	require.Equal(t, progressUpdate.FinalBytes, testClaimV1.ClaimBytes)
}

func TestFailedProphecy(t *testing.T) {
	ctx, _, keeper, validatorAddresses, err := CreateTestKeepers(t, false, 0.6, nil, []int64{3, 3, 4})
	validator1Pow3 := validatorAddresses[0]
	validator2Pow3 := validatorAddresses[1]
	validator3Pow4 := validatorAddresses[2]
	testClaimV1 := types.CreateTestClaimForValidator(validator1Pow3)
	testClaimAltV2 := types.CreateAlternateTestClaimForValidator(validator2Pow3)
	testClaimAltV3 := types.CreateAlternateTestClaimForValidator(validator3Pow4)

	//Test first claim
	progressUpdate, err := keeper.ProcessClaim(ctx, testClaimV1)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test second disagreeing claim processed fine
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimAltV2)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.SuccessStatus)
	require.Equal(t, progressUpdate.FinalBytes, testClaimV1.ClaimBytes)

	//Test third disagreeing claim processed fine and prophecy fails
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimAltV3)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.FailedStatus)
	require.Equal(t, progressUpdate.FinalBytes, nil)
}

func TestPower(t *testing.T) {
	//Testing with 3 validators but one has high enough power to overrule
	ctx, _, keeper, validatorAddresses, err := CreateTestKeepers(t, false, 0.7, nil, []int64{3, 7})
	validator1Pow3 := validatorAddresses[0]
	validator2Pow7 := validatorAddresses[1]
	testClaimV1 := types.CreateTestClaimForValidator(validator1Pow3)
	testClaimAltV2 := types.CreateAlternateTestClaimForValidator(validator2Pow7)

	//Test first claim
	progressUpdate, err := keeper.ProcessClaim(ctx, testClaimV1)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test second disagreeing claim processed fine and finalized to its bytes
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimAltV2)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.SuccessStatus)
	require.Equal(t, progressUpdate.FinalBytes, testClaimAltV2.ClaimBytes)

	//Test alternate power setup with validators of 5/4/3/9 and total power 22 and 12/21 required
	ctx, _, keeper, validatorAddresses, err = CreateTestKeepers(t, false, 0.571, nil, []int64{5, 4, 3, 9})
	validator1Pow5 := validatorAddresses[0]
	validator2Pow4 := validatorAddresses[1]
	validator3Pow3 := validatorAddresses[2]
	validator4Pow9 := validatorAddresses[3]
	testClaimV1 = types.CreateTestClaimForValidator(validator1Pow5)
	testClaimV2 := types.CreateTestClaimForValidator(validator2Pow4)
	testClaimV3 := types.CreateTestClaimForValidator(validator3Pow3)
	testClaimAltV4 := types.CreateAlternateTestClaimForValidator(validator4Pow9)

	//Test claim by v1
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimV1)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test claim by v2
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimV2)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test alternate claim by v4
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimAltV4)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test finalclaim by v1
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaimV3)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.SuccessStatus)
	require.Equal(t, progressUpdate.FinalBytes, testClaimAltV2.ClaimBytes)
}

func TestMultipleProphecies(t *testing.T) {
	//Test multiple prophecies running in parallel work fine as expected
	ctx, _, keeper, validatorAddresses, err := CreateTestKeepers(t, false, 0.7, nil, []int64{3, 7})
	validator1Pow3 := validatorAddresses[0]
	validator2Pow7 := validatorAddresses[1]
	testClaim1V1 := types.CreateTestClaimForValidator(validator1Pow3)
	testClaim1V2 := types.CreateTestClaimForValidator(validator2Pow7)
	secondTestClaim2V1 := types.NewClaim("oracleID2", validator1Pow3, []byte(types.AlternateTestByteString))
	secondTestClaim2V2 := types.NewClaim("oracleID2", validator2Pow7, []byte(types.AlternateTestByteString))

	//Test claim on first id with first validator
	progressUpdate, err := keeper.ProcessClaim(ctx, testClaim1V1)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.PendingStatus)

	//Test claim on second id with second validator
	progressUpdate, err = keeper.ProcessClaim(ctx, secondTestClaim2V2)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.SuccessStatus)
	require.Equal(t, progressUpdate.FinalBytes, secondTestClaim2V2.ClaimBytes)

	//Test claim on first id with first validator
	progressUpdate, err = keeper.ProcessClaim(ctx, testClaim1V2)
	require.NoError(t, err)
	require.Equal(t, progressUpdate.Status, types.SuccessStatus)
	require.Equal(t, progressUpdate.FinalBytes, testClaim1V1.ClaimBytes)

	//Test claim on second id with first validator
	progressUpdate, err = keeper.ProcessClaim(ctx, secondTestClaim2V1)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "Prophecy already finalized"))
}

func TestNonValidator(t *testing.T) {
	//TODO: anything from User that is not actually a validator fails
	err := errors.New("not yet implemented")
	require.NoError(t, err)
}
