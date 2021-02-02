package token_test

import (
	"testing"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/support/mock"
)

func TestExports(t *testing.T) {
	mock.CheckActorExports(t, token.Actor{})
}
