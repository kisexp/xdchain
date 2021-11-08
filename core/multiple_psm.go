package core

import (
	"context"
	"fmt"

	"github.com/kisexp/xdchain/common"
	"github.com/kisexp/xdchain/core/mps"
	"github.com/kisexp/xdchain/core/rawdb"
	"github.com/kisexp/xdchain/core/state"
	"github.com/kisexp/xdchain/core/types"
	"github.com/kisexp/xdchain/ethdb"
	"github.com/kisexp/xdchain/rpc"
	"github.com/kisexp/xdchain/trie"
)

type MultiplePrivateStateManager struct {
	// Low level persistent database to store final content in
	db                     ethdb.Database
	privateStatesTrieCache state.Database

	residentGroupByKey map[string]*mps.PrivateStateMetadata
	privacyGroupById   map[types.PrivateStateIdentifier]*mps.PrivateStateMetadata
}

func newMultiplePrivateStateManager(db ethdb.Database, config *trie.Config, residentGroupByKey map[string]*mps.PrivateStateMetadata, privacyGroupById map[types.PrivateStateIdentifier]*mps.PrivateStateMetadata) (*MultiplePrivateStateManager, error) {
	return &MultiplePrivateStateManager{
		db:                     db,
		privateStatesTrieCache: state.NewDatabaseWithConfig(db, config),
		residentGroupByKey:     residentGroupByKey,
		privacyGroupById:       privacyGroupById,
	}, nil
}

func (m *MultiplePrivateStateManager) StateRepository(blockHash common.Hash) (mps.PrivateStateRepository, error) {
	privateStatesTrieRoot := rawdb.GetPrivateStatesTrieRoot(m.db, blockHash)
	return mps.NewMultiplePrivateStateRepository(m.db, m.privateStatesTrieCache, privateStatesTrieRoot)
}

func (m *MultiplePrivateStateManager) ResolveForManagedParty(managedParty string) (*mps.PrivateStateMetadata, error) {
	psm, found := m.residentGroupByKey[managedParty]
	if !found {
		return nil, fmt.Errorf("unable to find private state metadata for managed party %s", managedParty)
	}
	return psm, nil
}

func (m *MultiplePrivateStateManager) ResolveForUserContext(ctx context.Context) (*mps.PrivateStateMetadata, error) {
	psi, ok := rpc.PrivateStateIdentifierFromContext(ctx)
	if !ok {
		psi = types.DefaultPrivateStateIdentifier
	}
	psm, found := m.privacyGroupById[psi]
	if !found {
		return nil, fmt.Errorf("unable to find private state for context psi %s", psi)
	}
	return psm, nil
}

func (m *MultiplePrivateStateManager) PSIs() []types.PrivateStateIdentifier {
	psis := make([]types.PrivateStateIdentifier, 0, len(m.privacyGroupById))
	for psi := range m.privacyGroupById {
		psis = append(psis, psi)
	}
	return psis
}

func (m *MultiplePrivateStateManager) NotIncludeAny(psm *mps.PrivateStateMetadata, managedParties ...string) bool {
	return psm.NotIncludeAny(managedParties...)
}

func (m *MultiplePrivateStateManager) CheckAt(root common.Hash) error {
	_, err := state.New(rawdb.GetPrivateStatesTrieRoot(m.db, root), m.privateStatesTrieCache, nil)
	return err
}

func (m *MultiplePrivateStateManager) TrieDB() *trie.Database {
	return m.privateStatesTrieCache.TrieDB()
}
