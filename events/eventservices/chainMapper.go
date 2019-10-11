package eventservices

import (
	"github.com/FactomProject/factomd/common/interfaces"
	"github.com/FactomProject/factomd/common/messages"
	"github.com/FactomProject/factomd/events/eventmessages/generated/eventmessages"
)

func mapCommitChain(entityState eventmessages.EntityState, msg interfaces.IMsg) *eventmessages.FactomEvent_ChainCommit {
	commitChainMsg, ok := msg.(*messages.CommitChainMsg)
	if !ok {
		return nil
	}
	commitChain := commitChainMsg.CommitChain
	ecPubKey := commitChain.ECPubKey.Fixed()
	sig := commitChain.Sig

	result := &eventmessages.FactomEvent_ChainCommit{
		ChainCommit: &eventmessages.ChainCommit{
			EntityState: entityState,
			ChainIDHash: &eventmessages.Hash{
				HashValue: commitChain.ChainIDHash.Bytes(),
			},
			EntryHash: &eventmessages.Hash{
				HashValue: commitChain.EntryHash.Bytes(),
			},
			Timestamp:            convertByteSlice6ToTimestamp(commitChain.MilliTime),
			Credits:              uint32(commitChain.Credits),
			EntryCreditPublicKey: ecPubKey[:],
			Signature:            sig[:],
			Version:              uint32(commitChain.Version),
			Weld: &eventmessages.Hash{
				HashValue: commitChain.Weld.Bytes(),
			},
		}}
	return result
}

func mapCommitChainState(state eventmessages.EntityState, msg interfaces.IMsg) *eventmessages.FactomEvent_StateChange {
	commitChainMsg, ok := msg.(*messages.CommitChainMsg)
	if !ok {
		return nil
	}
	result := &eventmessages.FactomEvent_StateChange{
		StateChange: &eventmessages.StateChange{
			EntityHash: &eventmessages.Hash{
				HashValue: commitChainMsg.CommitChain.ChainIDHash.Bytes()},
			EntityState: state,
		},
	}
	return result
}
